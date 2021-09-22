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
	rpcConfigPath    = "../.docker/rpc/"
	templateDataFile = "template.data.yml"
	singleNodeName   = "single"
)

var (
	goTemplateFile    = flag.String("go-template", "", "configuration template file for Go node")
	goDB              = flag.String("go-db", "leveldb", "database for Go node")
	sharpTemplateFile = flag.String("sharp-template", "", "configuration template file for C# node")
	sharpDB           = flag.String("sharp-db", "LevelDBStore", "database for C# node")
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
		err := convertTemplateToPlain(templateFile, tempDir, 4)
		if err != nil {
			log.Fatalf("failed to call ytt for Go template: %v", err)
		}
		err = generateGoConfig(tempDir+"/"+templateFile, *goDB, "")
		if err != nil {
			log.Fatalf("failed to generate Go configurations: %v", err)
		}

		err = convertTemplateToPlain(templateFile, tempDir, 7)
		if err != nil {
			log.Fatalf("failed to call ytt for Go template: %v", err)
		}
		err = generateGoConfig(tempDir+"/"+templateFile, *goDB, ".7")
		if err != nil {
			log.Fatalf("failed to generate Go configurations: %v", err)
		}
	}
	if templateFile := *sharpTemplateFile; templateFile != "" {
		err := convertTemplateToPlain(templateFile, tempDir, 4)
		if err != nil {
			log.Fatalf("failed to call ytt for C# template: %v", err)
		}
		err = generateSharpConfig(tempDir+"/"+templateFile, *sharpDB, "")
		if err != nil {
			log.Fatalf("failed to generate C# configurations: %v", err)
		}

		err = convertTemplateToPlain(templateFile, tempDir, 7)
		if err != nil {
			log.Fatalf("failed to call ytt for C# template: %v", err)
		}
		err = generateSharpConfig(tempDir+"/"+templateFile, *sharpDB, ".7")
		if err != nil {
			log.Fatalf("failed to generate C# configurations: %v", err)
		}
	}
}

func convertTemplateToPlain(templatePath string, tempDir string, nodeCount int) error {
	filePath := configPath + templatePath
	dataPath := configPath + templateDataFile
	cmd := template.NewCmd(template.NewOptions())
	cmd.SetArgs([]string{"-f", filePath, "-f", dataPath, "--output-files", tempDir,
		"--data-value-yaml", "validators_count=" + strconv.FormatInt(int64(nodeCount), 10)})
	err := cmd.Execute()
	if err != nil {
		return err
	}
	return nil
}

func generateGoConfig(templatePath, database, suffix string) error {
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
		template.ApplicationConfiguration.DBConfiguration.Type = database
		var configFile string
		nodeName, err := nodeNameFromSeedList(template.ApplicationConfiguration.NodePort, template.ProtocolConfiguration.SeedList)
		if err != nil {
			// it's an RPC node then
			configFile = rpcConfigPath + "go.protocol" + suffix + ".yml"
			template.ApplicationConfiguration.UnlockWallet.Path = ""
		} else {
			configFile = configPath + "go.protocol.privnet." + nodeName + suffix + ".yml"
		}
		bytes, err := yaml.Marshal(template)
		if err != nil {
			return fmt.Errorf("could not marshal config for node #%s: %v", nodeName, err)
		}
		err = ioutil.WriteFile(configFile, bytes, 0644)
		if err != nil {
			return fmt.Errorf("could not write config for node #%s: %v", nodeName, err)
		}
	}
	return nil
}

func generateSharpConfig(templatePath, storageEngine, suffix string) error {
	f, err := os.Open(templatePath)
	if err != nil {
		return fmt.Errorf("failed to open template: %v", err)
	}
	defer f.Close()
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
		template.ApplicationConfiguration.Storage.Engine = storageEngine
		var configFile string
		nodeName, err := nodeNameFromSeedList(template.ApplicationConfiguration.P2P.Port, template.ProtocolConfiguration.SeedList)
		if err != nil {
			// it's an RPC node then
			configFile = rpcConfigPath + "sharp.config" + suffix + ".json"
			template.ApplicationConfiguration.UnlockWallet = UnlockWallet{}

		} else {
			configFile = configPath + "sharp.config." + nodeName + suffix + ".json"
		}
		err = writeJSON(configFile, SharpConfig{
			ApplicationConfiguration: template.ApplicationConfiguration,
			ProtocolConfiguration:    template.ProtocolConfiguration,
		})
		if err != nil {
			return fmt.Errorf("could not write JSON config file for node #%s: %v", nodeName, err)
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
