package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
)

import "github.com/bradfitz/slice"

type Deployments struct {
	Items []struct {
		Meta struct {
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
		} `json:"metadata"`
	} `json:"items"`
}

type Deployments1 struct {
	Items []Deployment `json:"items"`
}

type Deployment struct {
	Meta Metadata `json:"metadata"`
}

type Metadata struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

func main() {
	flag.Parse()

	search := flag.Arg(0)
	cmd := flag.Arg(1)
	context := os.Getenv("KUBECTL_PLUGINS_GLOBAL_FLAG_CONTEXT")

	// fmt.Println("FLAG!!!", search)
	// env := os.Environ()

	// for k, v := range env {
	// 	fmt.Printf("%s: %s\n", k, v)
	// }

	kubectl := os.Getenv("KUBECTL_PLUGINS_CALLER")
	// fmt.Println(kubectl)

	searchRegexp := regexp.MustCompile(fmt.Sprintf(".*%s.*", search))

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

	podsJSON, err2 := exec.Command(kubectl, "--context", context, "get", "--raw=/api/v1/pods").Output()

	// fmt.Printf("PODS %s", podsJSON)
	if err2 != nil {
		panic(err2)
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
		//fmt.Printf("pod %s, %s\n", pod.Meta.Name, pod.Meta.Namespace)

		if searchRegexp.MatchString(pod.Meta.Name) {
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
