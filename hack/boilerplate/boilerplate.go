/*
Copyright 2019 The Kubernetes Authors All rights reserved.

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

package main

import (
	"bufio"
	"flag"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

func createCommandFlags() *flag.FlagSet {
	flag.String("filenames", ".", "list of all files to check, all files if unspecified")
	_, filename, _, _ := runtime.Caller(0)
	rootdir, _ := filepath.Abs(path.Join(path.Dir(filename), "/../../"))
	flag.String("--rootdir", rootdir, "root directory to examine")
	defaultBoilerplateDir := path.Join(rootdir, "hack/boilerplate")
	flag.String("--boilerplate-dir", defaultBoilerplateDir, "")
	flag.Parse()
	return flag.CommandLine
}

func getRefs(f *flag.FlagSet) map[string][]string {
	refs := map[string][]string{}
	glob, _ := filepath.Glob(path.Join(f.Lookup("--boilerplate-dir").Value.String(), "boilerplate.*.txt"))
	for i := range glob {
		extension := strings.Split(path.Base(glob[i]), ".")[1]
		ref_file, _ := os.Open(glob[i])
		scan := bufio.NewScanner(ref_file)
		scan.Split(bufio.ScanLines)
		var ref []string
		for scan.Scan() {
			ref = append(ref, scan.Text())
		}
		ref_file.Close()
		refs[extension] = ref
	}
	return refs
}

func filePasses(filename string, refs map[string][]string, regexs map[string]*regexp.Regexp) bool {
	f, err := os.Open(filename)
	if err != nil {
		return false
	}
	data, _ := ioutil.ReadAll(f)
	f.Close()
	basename := path.Base(filename)
	extenstion := fileExtension(filename)
}

func fileExtension(filename string) string {
	_, ext := path.Split(filename)
	exts := strings.Split(path.Base(ext), ".")
	return exts[len(exts)]
}

func main() {
	getRefs(createCommandFlags())
}

