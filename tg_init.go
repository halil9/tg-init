package main

import (
    "log"
    "os"
	"flag"
	"fmt"
	"path"
	"path/filepath"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

type conf struct {
	Directory struct {
		BasePath	string `yaml:"basePath"`
		ParentPaths []string `yaml:"parentPaths"`
		AppPaths 	[]string `yaml:"appPaths"`
	}
	Terragrunt struct {
		Main struct {
			RemoteState struct {
				BucketType	  string `yaml:"bucketType"`
				BucketName    string `yaml:"bucketName"`
				BucketKey     string `yaml:"bucketKey"`
				Region		  string `yaml:"region"`
				Encryption    bool   `yaml:"encryption"`
				DynamoDbTable string `yaml:"dynamoDbTable"`
			} `yaml:"remoteState"`
		} `yaml:"main"`
	}
}

func main() {

	file := flag.String("f", "", "a string yaml file for all variables.")

	flag.Parse()
	fmt.Println("Parsing YAML file")

	if *file == "" {
        fmt.Println("Please provide yaml file by using -f option")
        return
    }
	// Load the file;
	f, err := ioutil.ReadFile(*file)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	// Unmarshal our input YAML file into conf (var yamlConf)
	var yamlConf conf
	if err := yaml.Unmarshal(f, &yamlConf); err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
	// create basePath
	pwd, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
	}
	basePath := yamlConf.Directory.BasePath
	os.Mkdir(basePath, os.ModePerm)
	path.Join(pwd, basePath)
	os.Chdir(basePath)
	//parentPaths and app path variables
	parentPath := yamlConf.Directory.ParentPaths
	appPath    := yamlConf.Directory.AppPaths
	// parsed variables
	bucketType    := yamlConf.Terragrunt.Main.RemoteState.BucketType
	bucketName    := yamlConf.Terragrunt.Main.RemoteState.BucketName
	bucketKey     := yamlConf.Terragrunt.Main.RemoteState.BucketKey
	region		  := yamlConf.Terragrunt.Main.RemoteState.Region
	encryption    := yamlConf.Terragrunt.Main.RemoteState.Encryption
	dynamoDbTable := yamlConf.Terragrunt.Main.RemoteState.DynamoDbTable

	main_hcl_generate(basePath, bucketType, bucketName, bucketKey, region, encryption, dynamoDbTable)
	account_region_hcl_generate(parentPath)
	apps_hcl_generate(appPath, parentPath)
}

func main_hcl_generate(basePath string, bucketType string, bucketName string, keyName string, region string, encryption bool, DynamoDbTable string) {

	// create new empty hcl file object.
	f := hclwrite.NewEmptyFile()

	// create new file on system
	tfFile, err := os.Create("terragrunt.hcl")
	path.Join(basePath, "terragrunt.hcl")
	if err != nil {
		fmt.Println(err)
		return
	}
	// initialize the body of the new file object.
	rootBody := f.Body()
	// initilize the body for remote state for main hcl file.
	reqStateBlock := rootBody.AppendNewBlock("remote_state", nil)
	reqStateBlockBody := reqStateBlock.Body()
	// initilize remote state variables for main hcl file...
	reqStateBlockBody.SetAttributeValue("backend", cty.StringVal(bucketType))
	reqStateBlockBody.SetAttributeValue("config", cty.ObjectVal(map[string]cty.Value{
		"bucket":  cty.StringVal(bucketName),
		"key": cty.StringVal(keyName),
		"region": cty.StringVal(region),
		"encrypt": cty.BoolVal(encryption),
		"dynamodb_table": cty.StringVal(DynamoDbTable),
	}))
	fmt.Printf("%s", f.Bytes())
	tfFile.Write(f.Bytes())
}
func account_region_hcl_generate(parentPath []string){

	// join terragrunt files to region and envs directories.
	for _, p := range parentPath {
		os.MkdirAll(p, os.ModePerm)
		dir, parent := filepath.Split(p)
		fmt.Printf("Input: %q\n\tbaseDir: %q\n\tparentDir: %q\n", p + "/", dir, dir + parent)
		fullPath := path.Join(p, "account.hcl")
		parentPath := path.Join(dir, "account.hcl")
		// create new empty hcl file object.
		f := hclwrite.NewEmptyFile()
		// create new file on system
		fullPathtfFile, err := os.Create(fullPath)
		parentPathfFile, err := os.Create(parentPath)
		if err != nil {
			fmt.Println(err)
			return
		}
		rootBody := f.Body()
		localsBlock := rootBody.AppendNewBlock("locals", nil)
		localsBlock.Body()
		fmt.Printf("%s", f.Bytes())
		fullPathtfFile.Write(f.Bytes())
		parentPathfFile.Write(f.Bytes())
	}
}

func apps_hcl_generate(appPath []string, parentPath []string) {

	for _, a := range appPath {
		for _, p := range parentPath {
			path := filepath.Join(p, a)
			os.MkdirAll(path, os.ModePerm)
			hclFilePath := filepath.Join(path, "app.hcl")
			// create new empty hcl file object.
			f := hclwrite.NewEmptyFile()
			// create new file on system
			apptfFile, err := os.Create(hclFilePath)
			if err != nil {
				fmt.Println(err)
				return
			}
			rootBody := f.Body()
			includeBlock := rootBody.AppendNewBlock("include", []string{"root"})
			includeBlockBody := includeBlock.Body()
			pathBody := hclwrite.Tokens{
				{Type: hclsyntax.TokenIdent, Bytes: []byte(`find_in_parent_folders()`)},
			}
			includeBlockBody.SetAttributeRaw("path", pathBody)
			rootBody.AppendNewline()
			rootBody.SetAttributeValue("inputs", cty.ObjectVal(map[string]cty.Value{
				"instance_class":  cty.StringVal("asdasd"),
			}))
			fmt.Printf("%s", f.Bytes())
			apptfFile.Write(f.Bytes())
		}
	}
}
