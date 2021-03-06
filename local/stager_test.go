package local_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"

	docker "github.com/docker/docker/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	. "github.com/sclevine/cflocal/local"
	"github.com/sclevine/cflocal/mocks"
	"github.com/sclevine/cflocal/service"
	"github.com/sclevine/cflocal/utils"
)

var _ = Describe("Stager", func() {
	var (
		stager *Stager
		mockUI *mocks.MockUI
		logs   *gbytes.Buffer
	)

	BeforeEach(func() {
		mockUI = mocks.NewMockUI()

		client, err := docker.NewEnvClient()
		Expect(err).NotTo(HaveOccurred())
		client.UpdateClientVersion("")
		logs = gbytes.NewBuffer()
		stager = &Stager{
			UI:           mockUI,
			DiegoVersion: "0.1482.0",
			GoVersion:    "1.7",
			StackVersion: "1.86.0",
			Docker:       client,
			Logs:         io.MultiWriter(logs, GinkgoWriter),
		}
	})

	Describe("#Stage", func() {
		It("should return a droplet of a staged app", func() {
			appFileContents := bytes.NewBufferString("some-contents")
			appTar, err := utils.TarFile("some-file", appFileContents, int64(appFileContents.Len()), 0644)
			Expect(err).NotTo(HaveOccurred())
			droplet, err := stager.Stage(&StageConfig{
				AppTar:     appTar,
				Buildpacks: []string{"https://github.com/sclevine/cflocal-buildpack#v0.0.2"},
				AppConfig: &AppConfig{
					Name: "some-app",
					StagingEnv: map[string]string{
						"TEST_STAGING_ENV_KEY": "test-staging-env-value",
						"MEMORY_LIMIT":         "256m",
					},
					RunningEnv: map[string]string{
						"SOME_NA_KEY": "some-na-value",
					},
					Env: map[string]string{
						"TEST_ENV_KEY": "test-env-value",
						"MEMORY_LIMIT": "1024m",
					},
					Services: service.Services{
						"some-type": {{
							Name: "some-name",
						}},
					},
				},
			}, percentColor)
			Expect(err).NotTo(HaveOccurred())
			defer droplet.Close()

			Expect(logs.Contents()).To(MatchRegexp(`\[some-app\] % \S+ Compile message from stderr\.`))
			Expect(logs).To(gbytes.Say(`\[some-app\] % \S+ Compile arguments: /tmp/app /tmp/cache`))
			Expect(logs).To(gbytes.Say(`\[some-app\] % \S+ Compile message from stdout\.`))

			Expect(droplet.Size).To(BeNumerically(">", 500))
			Expect(droplet.Size).To(BeNumerically("<", 1500))

			dropletTar, err := gzip.NewReader(droplet)
			Expect(err).NotTo(HaveOccurred())
			dropletBuffer, err := ioutil.ReadAll(dropletTar)
			Expect(err).NotTo(HaveOccurred())

			file1, header1 := fileFromTar("./app/some-file", dropletBuffer)
			Expect(file1).To(Equal("some-contents"))
			Expect(header1.Uid).To(Equal(2000))
			Expect(header1.Gid).To(Equal(2000))

			file2, header2 := fileFromTar("./staging_info.yml", dropletBuffer)
			Expect(file2).To(ContainSubstring("start_command"))
			Expect(header2.Uid).To(Equal(2000))
			Expect(header2.Gid).To(Equal(2000))

			file3, header3 := fileFromTar("./app/env", dropletBuffer)
			Expect(file3).To(Equal(stagingEnvFixture))
			Expect(header3.Uid).To(Equal(2000))
			Expect(header3.Gid).To(Equal(2000))

			// TODO: test that no "some-app-staging-GUID" containers exist
			// TODO: test that termination via ExitChan works
			// TODO: test skipping detection when only one buildpack
			// TODO: test UI loading call
		})

		Context("on failure", func() {
			// TODO: test failure cases using reverse proxy
		})
	})

	Describe("#Download", func() {
		It("should return the specified file", func() {
			launcher, err := stager.Download("/tmp/lifecycle/launcher")
			Expect(err).NotTo(HaveOccurred())
			defer launcher.Close()

			Expect(launcher.Size).To(Equal(int64(3053594)))

			launcherBytes, err := ioutil.ReadAll(launcher)
			Expect(err).NotTo(HaveOccurred())
			Expect(launcherBytes).To(HaveLen(3053594))

			// TODO: test that no "some-app-launcher-GUID" containers exist
			// TODO: test UI loading call
		})

		Context("on failure", func() {
			// TODO: test failure cases using reverse proxy
		})
	})

	Describe("Dockerfile", func() {
		// TODO: test docker image via docker info?
	})
})

func fileFromTar(path string, tarball []byte) (string, *tar.Header) {
	file, header, err := utils.FileFromTar(path, bytes.NewReader(tarball))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	contents, err := ioutil.ReadAll(file)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	return string(contents), header
}
