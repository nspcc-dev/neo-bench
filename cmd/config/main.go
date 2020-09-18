package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/k14s/ytt/pkg/cmd/template"
	"github.com/nspcc-dev/neo-go/pkg/config"
	"gopkg.in/yaml.v2"
)

const (
	configPath       = "../.docker/ir/"
	templateDataFile = "template.data.yml"
	singleNodeName   = "single"
)

var (
	goTemplateFile    = flag.String("go-template", "", "configuration template file for Go node")
	sharpTemplateFile = flag.String("sharp-template", "", "configuration template file for C# node")
)

func main() {
	flag.Parse()

	tempDir, err := ioutil.TempDir("./", "")
	if err != nil {
		log.Fatalf("failed to create temporary directory: %v", err)
	}
	defer func() {
		err := os.RemoveAll(tempDir)
		if err != nil {
			log.Fatalf("failed to remove temporary directory: %v", err)
		}
	}()

	if templateFile := *goTemplateFile; templateFile != "" {
		err := convertTemplateToPlain(templateFile, tempDir)
		if err != nil {
			log.Fatalf("failed to call ytt for Go template: %v", err)
		}
		err = generateGoConfig(tempDir + "/" + templateFile)
		if err != nil {
			log.Fatalf("failed to generate Go configurations: %v", err)
		}
	}
	if templateFile := *sharpTemplateFile; templateFile != "" {
		err := convertTemplateToPlain(templateFile, tempDir)
		if err != nil {
			log.Fatalf("failed to call ytt for C# template: %v", err)
		}
		err = generateSharpConfig(tempDir + "/" + templateFile)
		if err != nil {
			log.Fatalf("failed to generate C# configurations: %v", err)
		}
	}
}

func convertTemplateToPlain(templatePath string, tempDir string) error {
	filePath := configPath + templatePath
	dataPath := configPath + templateDataFile
	cmd := template.NewCmd(template.NewOptions())
	cmd.SetArgs([]string{"-f", filePath, "-f", dataPath, "--output-files", tempDir})
	err := cmd.Execute()
	if err != nil {
		return err
	}
	return nil
}

func generateGoConfig(templatePath string) error {
	f, err := os.Open(templatePath)
	if err != nil {
		return fmt.Errorf("failed to open template: %v", err)
	}
	defer f.Close()
	decoder := yaml.NewDecoder(bufio.NewReader(f))
	for i := 0; ; i++ {
		var template config.Config
		err := decoder.Decode(&template)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("unable to decode node template #%d: %v", i, err)
		}
		nodeName, err := nodeNameFromSeedList(template.ApplicationConfiguration.NodePort, template.ProtocolConfiguration.SeedList)
		if err != nil {
			return err
		}
		bytes, err := yaml.Marshal(template)
		if err != nil {
			return fmt.Errorf("could not marshal config for node #%s: %v", nodeName, err)
		}
		err = ioutil.WriteFile(configPath+"go.protocol.privnet."+nodeName+".yml", bytes, 0644)
		if err != nil {
			return fmt.Errorf("could not write config for node #%s: %v", nodeName, err)
		}
	}
	return nil
}

func generateSharpConfig(templatePath string) error {
	f, err := os.Open(templatePath)
	if err != nil {
		return fmt.Errorf("failed to open template: %v", err)
	}
	defer f.Close()
	protocols := map[string]SharpProtocol{}
	decoder := yaml.NewDecoder(bufio.NewReader(f))
	for i := 0; ; i++ {
		var template SharpTemplate
		err := decoder.Decode(&template)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("unable to decode node template #%d: %v", i, err)
		}
		nodeName, err := nodeNameFromSeedList(template.ApplicationConfiguration.P2P.Port, template.ProtocolConfiguration.SeedList)
		if err != nil {
			return err
		}
		if nodeName == singleNodeName {
			protocols["sharp.protocol.single.json"] = SharpProtocol{
				ProtocolConfiguration: template.ProtocolConfiguration,
			}
		} else {
			protocols["sharp.protocol.json"] = SharpProtocol{
				ProtocolConfiguration: template.ProtocolConfiguration,
			}
		}
		path := configPath + "sharp.config." + nodeName + ".json"
		err = writeJSON(path, SharpConfig{
			ApplicationConfiguration: template.ApplicationConfiguration,
		})
		if err != nil {
			return fmt.Errorf("could not write JSON config file for node #%s: %v", nodeName, err)
		}
	}
	for protocolName, protocol := range protocols {
		err := writeJSON(configPath+protocolName, protocol)
		if err != nil {
			return fmt.Errorf("could not write JSON protocol file %s: %v", protocolName, err)
		}
	}
	return nil
}

func writeJSON(path string, obj interface{}) error {
	bytes, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path, bytes, 0644)
	if err != nil {
		return err
	}
	return nil
}

func nodeNameFromSeedList(port uint16, seedList []string) (string, error) {
	suffix := ":" + strconv.Itoa(int(port))
	for _, seed := range seedList {
		if strings.HasSuffix(seed, suffix) {
			node := strings.TrimSuffix(seed, suffix)
			if node == "node" {
				return singleNodeName, nil
			} else {
				return strings.TrimPrefix(node, "node_"), nil
			}
		}
	}
	return "", fmt.Errorf("node with port %v is not in the seed list", port)
}
