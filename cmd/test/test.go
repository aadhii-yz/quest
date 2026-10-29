package main

import (
	"fmt"
	"log"
	"os"

	cc "github.com/aadhii-yz/quest/internal/codecrafters"
	"gopkg.in/yaml.v3"
)

func main() {
	data, err := os.ReadFile("temp/build-your-own-grep/course-definition.yml")
	if err != nil {
		log.Fatalf("error reading file: %s", err.Error())
	}

	var cfg cc.CourseDefinition
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		log.Fatalf("error decoding course configuration file: %s", err.Error())
	}

	fmt.Printf("cfg.Stages: %v\n", cfg.Stages)
}
