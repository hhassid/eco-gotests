package tests

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/configmap"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/pod"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/reportxml"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/internal/querier"
	. "github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/internal/ranparam"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/ptp/internal/metrics"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/ptp/internal/profiles"
	ptpleap "github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/ptp/internal/ptp-leap"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/ptp/internal/ptpdaemon"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/ptp/internal/tsparams"
	"k8s.io/apimachinery/pkg/util/wait"
)

var _ = Describe("PTP Leap File", Label(tsparams.LabelLeapFile), func() {
	var prometheusAPI prometheusv1.API
	var leapConfigMap *configmap.Builder
	var err error

	BeforeEach(func() {
		By("creating a Prometheus API client")
		prometheusAPI, err = querier.CreatePrometheusAPIForCluster(RANConfig.Spoke1APIClient)
		Expect(err).ToNot(HaveOccurred(), "Failed to create Prometheus API client")

		By("ensuring clocks are locked before testing")
		err = metrics.AssertQuery(context.TODO(), prometheusAPI, metrics.ClockStateQuery{}, metrics.ClockStateLocked,
			metrics.AssertWithStableDuration(10*time.Second),
			metrics.AssertWithTimeout(5*time.Minute))
		Expect(err).ToNot(HaveOccurred(), "Failed to assert clock state is locked")
	})

	AfterEach(func() {
		By("restoring the original leap configmap")
		leapConfigMap, err = configmap.Pull(
			RANConfig.Spoke1APIClient, tsparams.LeapConfigmapName, ranparam.PtpOperatorNamespace)
		Expect(err).ToNot(HaveOccurred(), "Failed to pull original leap configmap")

		leapConfigMap.Definition.Data = map[string]string{}
		_, err = leapConfigMap.Update()
		Expect(err).ToNot(HaveOccurred(), "Failed to update original leap configmap")

		listPtpDaemonsetPods, err := pod.List(RANConfig.Spoke1APIClient, ranparam.PtpOperatorNamespace)
		Expect(err).ToNot(HaveOccurred(), "Failed to list PTP daemon set pods")
		for _, pod := range listPtpDaemonsetPods {
			_, err = pod.DeleteAndWait(5 * time.Minute)
			Expect(err).ToNot(HaveOccurred(), "Failed to delete PTP daemon set pod")
		}

		prometheusAPI, err = querier.CreatePrometheusAPIForCluster(RANConfig.Spoke1APIClient)
		Expect(err).ToNot(HaveOccurred(), "Failed to create Prometheus API client")

		By("ensuring clocks are locked after testing")
		err = metrics.AssertQuery(context.TODO(), prometheusAPI, metrics.ClockStateQuery{}, metrics.ClockStateLocked,
			metrics.AssertWithStableDuration(10*time.Second),
			metrics.AssertWithTimeout(5*time.Minute))
		Expect(err).ToNot(HaveOccurred(), "Failed to assert clock state is locked")
	})

	It("should add leap event announcement in leap configmap when removing the last announcement",
		reportxml.ID("75325"), func() {
			testRanAtLeastOnce := false

			By("pulling leap configmap")
			leapConfigMap, err = configmap.Pull(
				RANConfig.Spoke1APIClient, tsparams.LeapConfigmapName, ranparam.PtpOperatorNamespace)
			Expect(err).ToNot(HaveOccurred(), "Failed to pull leap configmap")

			nodeInfoMap, err := profiles.GetNodeInfoMap(RANConfig.Spoke1APIClient)
			Expect(err).ToNot(HaveOccurred(), "Failed to get node info map")

			for _, nodeInfo := range nodeInfoMap {
				By(fmt.Sprintf("removing the last leap announcement from the leap configmap for node %s", nodeInfo.Name))
				testRanAtLeastOnce = true
				withoutLastLeapAnnouncementData := ptpleap.RemoveLastLeapAnnouncement(leapConfigMap.Object.Data[nodeInfo.Name])
				_, err := leapConfigMap.WithData(
					map[string]string{nodeInfo.Name: withoutLastLeapAnnouncementData}).Update()
				Expect(err).ToNot(HaveOccurred(), "Failed to update original leap configmap")

				By("deleting the PTP daemon pod for node " + nodeInfo.Name)
				ptpDaemonPod, err := ptpdaemon.GetPtpDaemonPodOnNode(RANConfig.Spoke1APIClient, nodeInfo.Name)
				Expect(err).ToNot(HaveOccurred(), "Failed to get PTP daemon pod for node %s", nodeInfo.Name)

				_, err = ptpDaemonPod.Delete()
				Expect(err).ToNot(HaveOccurred(), "Failed to delete PTP daemon pod for node %s", nodeInfo.Name)

				// By("sleeping for 10 seconds")
				// time.Sleep(10 * time.Second)

				By("validating the PTP daemon pod is running on node " + nodeInfo.Name)
				err = ptpdaemon.ValidatePtpDaemonPodRunning(RANConfig.Spoke1APIClient, nodeInfo.Name)
				Expect(err).ToNot(HaveOccurred(), "Failed to validate PTP daemon pod running on node %s", nodeInfo.Name)
			}

			leapConfigMap, err := configmap.Pull(
				RANConfig.Spoke1APIClient, tsparams.LeapConfigmapName, ranparam.PtpOperatorNamespace)
			Expect(err).ToNot(HaveOccurred(), "Failed to pull leap configmap")

			By("waiting for configmap to be updated with today's date leap announcement")
			newLeapConfigMap, err := waitForConfigmapToBeUpdated(leapConfigMap, 5*time.Second, 10*time.Minute)
			Expect(err).ToNot(HaveOccurred(), "Failed to wait for configmap to be updated")

			By("ensuring new last announcement is different from the original last announcement")
			for _, nodeInfo := range nodeInfoMap {
				lastAnnouncement, err := ptpleap.GetLastAnnouncement(leapConfigMap.Object.Data[nodeInfo.Name])
				Expect(err).ToNot(HaveOccurred(), "Failed to get last announcement")
				newLastAnnouncement, err := ptpleap.GetLastAnnouncement(newLeapConfigMap.Object.Data[nodeInfo.Name])
				Expect(err).ToNot(HaveOccurred(), "Failed to get last announcement")
				Expect(newLastAnnouncement).NotTo(Equal(lastAnnouncement), "Last announcement should be different")
			}

			if !testRanAtLeastOnce {
				Skip("Could not find any node to update leap configmap")
			}
		})
})

// waitForConfigmapToBeUpdated waits until the configmap is updated with the last leap announcement line
// that matches today's date in UTC, formatted "d Mon yyyy".
func waitForConfigmapToBeUpdated(leapConfigMap *configmap.Builder,
	interval time.Duration,
	timeout time.Duration) (*configmap.Builder, error) {
	err := wait.PollUntilContextTimeout(
		context.TODO(), interval, timeout, true, func(ctx context.Context) (bool, error) {
			today := time.Now().UTC().Format("2 Jan 2006")

			leapConfigMap, err := configmap.Pull(
				RANConfig.Spoke1APIClient, tsparams.LeapConfigmapName, ranparam.PtpOperatorNamespace)
			if err != nil {
				return false, nil
			}

			for _, leapConfigmapData := range leapConfigMap.Object.Data {
				if strings.Contains(leapConfigmapData, today) {
					return true, nil
				}
			}

			return false, nil
		})
	if err != nil {
		return nil, err
	}

	return leapConfigMap, nil
}
