package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/onsi/gomega/format"
	"github.com/paketo-buildpacks/occam"
	"github.com/paketo-buildpacks/occam/packagers"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

var (
	buildpack          string
	phpDistBuildpack   string
	phpFpmBuildpack    string
	httpdBuildpack     string
	phpHttpdBuildpack  string
	nginxBuildpack     string
	phpNginxBuildpack  string
	watchexecBuildpack string
	root               string

	buildpackInfo struct {
		Buildpack struct {
			ID   string
			Name string
		}
	}
)

func TestIntegration(t *testing.T) {
	Expect := NewWithT(t).Expect

	format.MaxLength = 0

	var config struct {
		PhpDist   string `json:"php-dist"`
		PhpFpm    string `json:"php-fpm"`
		Httpd     string `json:"httpd"`
		PhpHttpd  string `json:"php-httpd"`
		Nginx     string `json:"nginx"`
		PhpNginx  string `json:"php-nginx"`
		Watchexec string `json:"watchexec"`
	}

	integrationFile, err := os.Open("../integration.json")
	Expect(err).NotTo(HaveOccurred())
	defer func() {
		if err := integrationFile.Close(); err != nil {
			Expect(err).NotTo(HaveOccurred())
		}
	}()

	Expect(json.NewDecoder(integrationFile).Decode(&config)).To(Succeed())

	buildpackFile, err := os.Open("../buildpack.toml")
	Expect(err).NotTo(HaveOccurred())

	_, err = toml.NewDecoder(buildpackFile).Decode(&buildpackInfo)
	Expect(err).NotTo(HaveOccurred())
	Expect(buildpackFile.Close()).To(Succeed())

	root, err = filepath.Abs("./..")
	Expect(err).ToNot(HaveOccurred())

	buildpackStore := occam.NewBuildpackStore()
	libpakBuildpackStore := occam.NewBuildpackStore().WithPackager(packagers.NewLibpak())
	targetedBuildpackStore := buildpackStore.WithTarget("linux/" + runtime.GOARCH)
	targetedLibpakBuildpackStore := libpakBuildpackStore.WithTarget("linux/" + runtime.GOARCH)

	buildpack, err = buildpackStore.Get.
		WithVersion("1.2.3").
		Execute(root)
	Expect(err).NotTo(HaveOccurred())

	phpDistBuildpack, err = targetedBuildpackStore.Get.
		Execute(config.PhpDist)
	Expect(err).NotTo(HaveOccurred())

	phpFpmBuildpack, err = targetedBuildpackStore.Get.
		Execute(config.PhpFpm)
	Expect(err).NotTo(HaveOccurred())

	httpdBuildpack, err = targetedBuildpackStore.Get.
		Execute(config.Httpd)
	Expect(err).NotTo(HaveOccurred())

	phpHttpdBuildpack, err = targetedBuildpackStore.Get.
		Execute(config.PhpHttpd)
	Expect(err).NotTo(HaveOccurred())

	nginxBuildpack, err = targetedBuildpackStore.Get.
		Execute(config.Nginx)
	Expect(err).NotTo(HaveOccurred())

	phpNginxBuildpack, err = targetedBuildpackStore.Get.
		Execute(config.PhpNginx)
	Expect(err).NotTo(HaveOccurred())

	watchexecBuildpack, err = targetedLibpakBuildpackStore.Get.
		Execute(config.Watchexec)
	Expect(err).NotTo(HaveOccurred())

	SetDefaultEventuallyTimeout(10 * time.Second)

	suite := spec.New("Integration", spec.Report(report.Terminal{}), spec.Parallel())
	suite("Httpd", testHttpd)
	suite("HttpdReload", testHttpdReload)
	suite("Nginx", testNginx)
	suite("NginxReload", testNginxReload)
	suite.Run(t)
}
