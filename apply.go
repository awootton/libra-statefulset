package librastatefulset

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/awootton/knotfreeiot/kubectl"
)

// LibraValidatorParams is
type LibraValidatorParams struct {
	// replaces MY_DOCKER_REPO_HERE
	DockerRepo   string // eg   gcr.io/fair-theater-238820/
	NodeCount    int
	LocalStorage string
	LibraPath    string
}

// NewLibraValidatorParams may need the user to go to ~/libra_secrets and edit libra-statefulset-params.json
func NewLibraValidatorParams() *LibraValidatorParams {
	params := &LibraValidatorParams{}
	params.DockerRepo = "gcr.io/fair-theater-238820"
	params.NodeCount = 3
	params.LocalStorage = "~/libra_secrets"
	params.LibraPath = "~/Documents/workspace/libra"
	return params
}

// CreateConfigsLocally will set up the necessary files for later
func CreateConfigsLocally(params *LibraValidatorParams) *LibraValidatorParams {

	foundParams := false
	if params == nil {
		params = NewLibraValidatorParams()
		// try to read it.
		str, err := ioutil.ReadFile(fixpath(params.LocalStorage) + "/" + "libra-statefulset-params.json")
		if err == nil {
			p := &LibraValidatorParams{}
			err = json.Unmarshal(str, p)
			if err == nil {
				params = p
				foundParams = true
			}
		}
	}
	params.LocalStorage = fixpath(params.LocalStorage)
	if _, err := os.Stat(params.LocalStorage); os.IsNotExist(err) {
		fmt.Println("no directory found at ", params.LocalStorage, "please initialize")
		panic("needs mkdir ~/libra_secrets")
	}
	// if not found saves params then make some
	if foundParams == false {
		fmt.Println("saving params to ", params.LocalStorage+"/"+"libra-statefulset-params.json")
		str, err := json.MarshalIndent(params, "", "    ")
		if err == nil {
			err = ioutil.WriteFile(params.LocalStorage+"/"+"libra-statefulset-params.json", []byte(str), 0644)
		}
	}
	// is there a $SHARED_SECRET? eg. 0123456789abcdef101112131415161718191a1b1c1d1e1f2021222324252627
	sharedSecretBytes, err := ioutil.ReadFile(params.LocalStorage + "/" + "shared_secret.txt")
	if err == nil || len(sharedSecretBytes) != 64 {
		fmt.Println("generating new shared secret")
		tmp := make([]byte, 32)
		rand.Read(tmp)
		sharedSecretBytes = []byte(hex.EncodeToString(tmp))
		ioutil.WriteFile(params.LocalStorage+"/"+"shared_secret.txt", sharedSecretBytes, 0644)
	}
	sharedSecret := string(sharedSecretBytes)

	params.LibraPath = fixpath(params.LibraPath)

	configBuilderPath := params.LibraPath + "/config/config-builder"
	_, err = ioutil.ReadFile(configBuilderPath + "/Cargo.toml")
	if err != nil {
		fmt.Println("missing a clone of the libra project at " + params.LibraPath)
		fmt.Println("expecting config-builder at  " + configBuilderPath)
		panic("quitting till libra")
	}

	// do you have have rust installed ? rtfm
	result, err := kubectl.K8s("cd "+configBuilderPath+" ; cargo build", "")
	fmt.Println("build got", result, err)

	configBuilderBinary := params.LibraPath + "/target/debug/config-builder"
	if _, err := os.Stat(configBuilderBinary); os.IsNotExist(err) {
		fmt.Println("no binary found at ", configBuilderBinary)
		panic("needs debug")
	}

	//now we can call it
	// see https://github.com/libra/libra/tree/master/config
	for i := 0; i < params.NodeCount; i++ {
		istr := strconv.FormatInt(int64(i), 10)
		address0 := "100.100.100.100"
		tmpstr := strconv.FormatInt(int64(i+100), 10)
		addressI := tmpstr + "." + tmpstr + "." + tmpstr + "." + tmpstr

		dest := params.LocalStorage + "/nodes/config" + istr
		kubectl.K("rm " + dest + "/*")
		kubectl.K("mkdir " + dest)

		cmd := configBuilderBinary + " validator "
		cmd += `-a "/ip4/` + addressI + `/tcp/6180" `
		cmd += `-b "/ip4/` + address0 + `/tcp/6180" `
		cmd += `-d /opt/libra/data   `
		cmd += `-i ` + istr + ` `
		cmd += `-l "/ip4/0.0.0.0/tcp/6180" `
		cmd += `-n ` + strconv.FormatInt(int64(params.NodeCount), 10) + ` `
		cmd += `-s ` + sharedSecret + ` `
		cmd += `-o ` + dest

		result, err := kubectl.K8s(cmd, "")
		fmt.Println("config-builder got", result, err)

		//and also the mint
		cmd = configBuilderBinary + " faucet "
		cmd += `-s ` + sharedSecret + ` `
		cmd += `-o ` + dest
		result, err = kubectl.K8s(cmd, "")
		fmt.Println("config-builder got", result, err)
	}

	_ = sharedSecret
	return params
}

// this is how I test:
// this: kind create cluster --config kind-example-config.yaml
// 		 kubectl config use-context "kind-kind"
// or this: gcloud container clusters get-credentials ...
// kubectl create ns kibra
// kubectl config set-context --current --namespace=libra

// Apply is to 	//make a stateful set
func Apply(params *LibraValidatorParams) {

	// before we do this we must create the configs
	params = CreateConfigsLocally(params)

	defer kubectl.K("rm tmp.yaml")

	kubectl.K("pwd")                     // /Users/awootton/Documents/workspace/libra-statefulset
	kubectl.K("kubectl create ns libra") // ok if error 2nd time.
	kubectl.K("kubectl config set-context --current --namespace=libra")

	bytes1, _ := ioutil.ReadFile(params.LocalStorage + "/nodes/config0/mint.key")
	bytes2, _ := ioutil.ReadFile(params.LocalStorage + "/nodes/config1/mint.key")
	if bytes.Equal(bytes1, bytes2) == false {
		fmt.Println("the mints are all the same")
	}

	bytes1, _ = ioutil.ReadFile(params.LocalStorage + "/nodes/config0/genesis.blob")
	bytes2, _ = ioutil.ReadFile(params.LocalStorage + "/nodes/config1/genesis.blob")
	if bytes.Equal(bytes1, bytes2) == false {
		fmt.Println("the blobs are all the same")
	}
	// the seeds are all different and are referenced by the node.config

	// loop over the files and move to tmp
	kubectl.K("cd tmp;rm *")
	for i := 0; i < params.NodeCount; i++ {
		istr := strconv.FormatInt(int64(i), 10)
		path := params.LocalStorage + "/nodes/config" + istr
		// concat together all the configs into a flat dir
		// and replace the ip's
		kubectl.K("rm " + path + "/.DS_Store")
		cmd := "cp -R " + path + "/ tmp/"
		kubectl.K(cmd)
		// now rename the blob config.
		cmd = "mv  " + "tmp/node.config.toml " + "tmp/" + istr + "node.config.toml "
		kubectl.K(cmd)
	}

	kubectl.K("cp startup.sh tmp/startup.sh")

	kubectl.K("kubectl delete cm libra-config-map")

	kubectl.K(`kubectl create configmap libra-config-map --from-file tmp/`)

	kubectl.K(`kubectl delete statefulset.apps/libra`) // may be necessary

	replacerList := make([]string, 0)
	replacerList = append(replacerList, "MY_DOCKER_REPO_HERE", params.DockerRepo)

	inKind := false
	val, _ := kubectl.K8s("kubectl get no ", "")
	if strings.Contains(val, "kind-control-plane") == true {
		// we're in kind
		inKind = true
		replacerList = append(replacerList, "libra-storage-class", "standard")
	}

	replacer := strings.NewReplacer(replacerList...)

	if !inKind {
		val, _ := kubectl.K8s("kubectl get storageclass ", "")
		fmt.Println("found storage classes", val)
		// we only want to do this once and never in kind.
		if strings.Contains(val, "libra-storage-class") == false {
			kubectl.Quiet = true
			//kubectl.K("kubectl apply -f libra-storage.yaml")
			KubeCtlApplyReplaced(replacer, "libra-storage.yaml")
			kubectl.Quiet = false
		}
	}

	KubeCtlApplyReplaced(replacer, "libra-validator.yaml")

	fmt.Println("when:", time.Now())
}

// KubeCtlApplyReplaced will do a kubectl apply
func KubeCtlApplyReplaced(replacer *strings.Replacer, fname string) {

	content, err := ioutil.ReadFile(fname)
	if err != nil {
		panic("failed to read file" + fname)
	}
	tmpfilecontents := replacer.Replace(string(content))
	//fmt.Println("will apply:", tmpfilecontents)
	err = ioutil.WriteFile("tmp.yaml", []byte(tmpfilecontents), 0644)
	kubectl.K("kubectl apply -f tmp.yaml")

}

func fixpath(path string) string {
	if strings.HasPrefix(path, "~") {
		s, _ := os.UserHomeDir()
		path = strings.Replace(path, "~", s, 1)
	}
	return path
}
