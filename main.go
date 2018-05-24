package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/bradfitz/slice"
)

func main() {
	flag.Parse()

	search := flag.Arg(0)
	cmd := flag.Arg(1)
	context := os.Getenv("KUBECTL_PLUGINS_GLOBAL_FLAG_CONTEXT")
	kubectl := os.Getenv("KUBECTL_PLUGINS_CALLER")

	type PodMeta struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	}

	type Pod struct {
		Meta PodMeta `json:"metadata"`
	}

	type Pods struct {
		Items []Pod `json:"items"`
	}

	podsJSON, getError := exec.Command(kubectl, "--context", context, "get", "--raw=/api/v1/pods").Output()

	if getError != nil {
		panic(getError)
	}

	var pods Pods

	jsonErr := json.Unmarshal(podsJSON, &pods)

	if jsonErr != nil {
		panic(jsonErr)
	}

	var targetPod string
	var targetNs string
	var matchingPods []Pod

	for _, pod := range pods.Items {
		if strings.Contains(pod.Meta.Name, search) {
			matchingPods = append(matchingPods, pod)
		}
	}

	if len(matchingPods) == 0 {
		fmt.Printf("Nothing found for %s", search)
		os.Exit(0)
	}

	slice.Sort(matchingPods[:], func(i, j int) bool {
		return matchingPods[i].Meta.Name < matchingPods[j].Meta.Name
	})

	fmt.Println("Pods matching the search:")
	for _, item := range matchingPods {
		fmt.Printf("%s in namespace %s\n", item.Meta.Name, item.Meta.Namespace)
	}

	targetPod = matchingPods[0].Meta.Name
	targetNs = matchingPods[0].Meta.Namespace

	fmt.Printf("\nAttaching to %s in namespace %s...\n", targetPod, targetNs)

	kubectlExec := exec.Command(kubectl, "--context", context, "exec", "-it", targetPod, "-n", targetNs, cmd)

	kubectlExec.Stdout = os.Stdout
	kubectlExec.Stdin = os.Stdin
	kubectlExec.Stderr = os.Stderr
	execErr := kubectlExec.Run()

	if execErr != nil {
		panic(execErr)
	}
}
