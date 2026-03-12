package tests

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/internal/querier"
	. "github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/ptp/internal/metrics"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/ptp/internal/tsparams"
)

var _ = Describe("PTP Event Consumer", Label(tsparams.LabelTestingConsumerEvents), func() {
	var (
		prometheusAPI prometheusv1.API
	)

	BeforeEach(func() {
		var err error

		By("creating a Prometheus API client")

		prometheusAPI, err = querier.CreatePrometheusAPIForCluster(RANConfig.Spoke1APIClient)
		Expect(err).ToNot(HaveOccurred(), "Failed to create Prometheus API client")

		By("ensuring clocks are locked before testing")

		err = metrics.EnsureClocksAreLocked(prometheusAPI)
		Expect(err).ToNot(HaveOccurred(), "Failed to assert clock state is locked")

		By("saving PtpConfigs before testing")
	})

	It("test consumer events", func() {
		By("sleeping for 1 minutes")
		time.Sleep(1 * time.Minute)
	})
})
