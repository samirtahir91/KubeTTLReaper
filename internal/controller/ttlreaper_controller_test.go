/*
Copyright 2024.

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

package controller

import (
	"fmt"
	"kubettlreaper/test/utils"
	"os"
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
)

var namespace = os.Getenv("OPERATOR_NAMESPACE")

// Function to initialise os vars
func init() {
	if namespace == "" {
		panic(fmt.Errorf("OPERATOR_NAMESPACE environment variable(s) not set"))
	}
}

var _ = Describe("TtlReaper Controller", Ordered, func() {

	BeforeAll(func() {
		By("removing manager namespace")
		cmd := exec.Command("kubectl", "delete", "ns", namespace)
		_, _ = utils.Run(cmd)

		By("creating manager namespace")
		err := utils.CreateNamespace(ctx, k8sClient, namespace)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterAll(func() {
		By("removing manager namespace")
		cmd := exec.Command("kubectl", "delete", "ns", namespace)
		_, _ = utils.Run(cmd)
	})

	Context("When creating the operator ConfigMap", func() {
		It("should successfully load the GVKs and check interval", func() {
			By("Creating the operator configMap with sample GVKs")
			configMap, err := utils.CreateConfigMap(ctx, k8sClient, utils.ConfigurationName, namespace, "5s")
			Expect(err).NotTo(HaveOccurred())
			Expect(configMap).NotTo(BeNil())
			Expect(configMap.Data).To(HaveKey("check-interval"))
			Expect(configMap.Data["gvk-list"]).To(ContainSubstring("group: \"\""))

			By("Waiting for the config success event to be recorded")
			err = utils.CheckEvent(
				ctx,
				k8sClient,
				utils.ConfigurationName,
				namespace,
				"Normal",
				"ValidConfig",
				"Processing GVKs from configMap",
			)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When creating a RoleBinding with a TTL of 10s", func() {
		roleBindingName := "master-chief"
		It("should exist with a TTL", func() {
			By("Creating the RoleBinding")
			err := utils.CreateRoleBinding(ctx, k8sClient, roleBindingName, namespace, "10s")
			Expect(err).NotTo(HaveOccurred())
		})
		It("should be deleted after 10s by the operator", func() {
			By("Waiting for the RoleBinding to be deleted")
			gvk := schema.GroupVersionKind{
				Group:   "rbac.authorization.k8s.io",
				Version: "v1",
				Kind:    "RoleBinding",
			}
			utils.WaitForDeleted(ctx, k8sClient, namespace, roleBindingName, gvk)
			By("Waiting for the ReapedOnTTL event to be recorded")
		})
	})

	Context("When creating a Secret with a TTL of 10s", func() {
		secretName := "master-chief"
		It("should exist with a TTL", func() {
			By("Creating the Secret")
			err := utils.CreateSecret(ctx, k8sClient, secretName, namespace, "10s")
			Expect(err).NotTo(HaveOccurred())
		})
		It("should be deleted after 10s by the operator", func() {
			By("Waiting for the Secret to be deleted")
			gvk := schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "Secret",
			}
			utils.WaitForDeleted(ctx, k8sClient, namespace, secretName, gvk)
			By("Waiting for the ReapedOnTTL event to be recorded")
		})
	})

})
