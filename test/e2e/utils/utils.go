// Copyright 2020 Intel Corporation. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/onsi/gomega"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	testutils "k8s.io/kubernetes/test/utils"
)

const (
	nodeListTimeout = 10 * time.Second
	poll            = time.Second
)

// WaitForNodesWithResource waits for nodes to have positive allocatable resource.
func WaitForNodesWithResource(c clientset.Interface, res v1.ResourceName, timeout time.Duration) error {
	framework.Logf("Waiting up to %s for any positive allocatable resource %q", timeout, res)
	start := time.Now()
	err := wait.Poll(poll, timeout,
		func() (bool, error) {
			for t := time.Now(); time.Since(t) < nodeListTimeout; time.Sleep(poll) {
				nodelist, err := c.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
				if err != nil {
					if testutils.IsRetryableAPIError(err) {
						continue
					}
					return false, err
				}

				resNum := 0
				for _, item := range nodelist.Items {
					if q, ok := item.Status.Allocatable[res]; ok {
						resNum = resNum + int(q.Value())
					}
				}
				framework.Logf("Found %d of %q. Elapsed: %s", resNum, res, time.Since(start))
				return resNum > 0, nil
			}

			return false, errors.New("unable to list nodes")
		})
	return err
}

// WaitForPodFailure waits for a pod to fail.
// This function used to be a part of k8s e2e framework, but was deleted in
// https://github.com/kubernetes/kubernetes/pull/86732.
func WaitForPodFailure(f *framework.Framework, name string, timeout time.Duration) {
	gomega.Expect(e2epod.WaitForPodCondition(f.ClientSet, f.Namespace.Name, name, "success or failure", timeout,
		func(pod *v1.Pod) (bool, error) {
			switch pod.Status.Phase {
			case v1.PodFailed:
				return true, nil
			case v1.PodSucceeded:
				return true, fmt.Errorf("pod %q successed with reason: %q, message: %q", name, pod.Status.Reason, pod.Status.Message)
			default:
				return false, nil
			}
		},
	)).To(gomega.Succeed(), "wait for pod %q to fail", name)
}

// LocateRepoFile locates a file inside this repository.
func LocateRepoFile(repopath string) (string, error) {
	root := os.Getenv("PLUGINS_REPO_DIR")
	if root != "" {
		path := filepath.Join(root, repopath)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			return path, nil
		}
	}

	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	path := filepath.Join(currentDir, repopath)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return path, nil
	}
	path = filepath.Join(currentDir, "../../"+repopath)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return path, err
	}

	return "", errors.New("no file found, try to define PLUGINS_REPO_DIR pointing to the root of the repository")
}
