/*
Copyright 2016 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package notify

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"k8s.io/minikube/pkg/minikube/config"
	"k8s.io/minikube/pkg/minikube/constants"
	"k8s.io/minikube/pkg/minikube/localpath"
	"k8s.io/minikube/pkg/minikube/out"
	"k8s.io/minikube/pkg/util/lock"
	"k8s.io/minikube/pkg/version"
)

var (
	timeLayout              = time.RFC1123
	lastUpdateCheckFilePath = localpath.MakeMiniPath("last_update_check")
)

// MaybePrintUpdateTextFromGithub prints update text if needed, from github
func MaybePrintUpdateTextFromGithub() bool {
	return MaybePrintUpdateText(constants.GithubMinikubeReleasesURL, lastUpdateCheckFilePath)
}

// MaybePrintUpdateText prints update text, returns a bool if life is good.
func MaybePrintUpdateText(url string, lastUpdatePath string) bool {
	if !shouldCheckURLVersion(lastUpdatePath) {
		return false
	}
	latestVersion, err := getLatestVersionFromURL(url)
	if err != nil {
		glog.Warning(err)
		return true
	}
	localVersion, err := version.GetSemverVersion()
	if err != nil {
		glog.Warning(err)
		return true
	}
	if localVersion.Compare(latestVersion) < 0 {
		if err := writeTimeToFile(lastUpdateCheckFilePath, time.Now().UTC()); err != nil {
			glog.Errorf("write time failed: %v", err)
		}
		url := "https://github.com/kubernetes/minikube/releases/tag/v" + latestVersion.String()
		out.ErrT(out.Celebrate, `minikube {{.version}} is available! Download it: {{.url}}`, out.V{"version": latestVersion, "url": url})
		out.ErrT(out.Tip, "To disable this notice, run: 'minikube config set WantUpdateNotification false'\n")
		return true
	}
	return false
}

func shouldCheckURLVersion(filePath string) bool {
	if !viper.GetBool(config.WantUpdateNotification) {
		return false
	}
	lastUpdateTime := getTimeFromFileIfExists(filePath)
	return time.Since(lastUpdateTime).Hours() >= viper.GetFloat64(config.ReminderWaitPeriodInHours)
}

// Release represents a release
type Release struct {
	Name      string
	Checksums map[string]string
}

// Releases represents several release
type Releases []Release

func getJSON(url string, target *Releases) error {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return errors.Wrap(err, "error creating new http request")
	}
	ua := fmt.Sprintf("Minikube/%s Minikube-OS/%s",
		version.GetVersion(), runtime.GOOS)

	req.Header.Set("User-Agent", ua)

	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrapf(err, "error with http GET for endpoint %s", url)
	}

	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(target)
}

func getLatestVersionFromURL(url string) (semver.Version, error) {
	r, err := GetAllVersionsFromURL(url)
	if err != nil {
		return semver.Version{}, err
	}
	return semver.Make(strings.TrimPrefix(r[0].Name, version.VersionPrefix))
}

// GetAllVersionsFromURL get all versions from a JSON URL
func GetAllVersionsFromURL(url string) (Releases, error) {
	var releases Releases
	glog.Info("Checking for updates...")
	if err := getJSON(url, &releases); err != nil {
		return releases, errors.Wrap(err, "Error getting json from minikube version url")
	}
	if len(releases) == 0 {
		return releases, errors.Errorf("There were no json releases at the url specified: %s", url)
	}
	return releases, nil
}

func writeTimeToFile(path string, inputTime time.Time) error {
	err := lock.WriteFile(path, []byte(inputTime.Format(timeLayout)), 0644)
	if err != nil {
		return errors.Wrap(err, "Error writing current update time to file: ")
	}
	return nil
}

func getTimeFromFileIfExists(path string) time.Time {
	lastUpdateCheckTime, err := ioutil.ReadFile(path)
	if err != nil {
		return time.Time{}
	}
	timeInFile, err := time.Parse(timeLayout, string(lastUpdateCheckTime))
	if err != nil {
		return time.Time{}
	}
	return timeInFile
}
