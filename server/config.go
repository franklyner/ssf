package server

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/spf13/viper"
)

// Config Central access point to all config properties
type Config struct {
	p map[string]string
}

// Get returns a cached property value. Panics if the property doesn't exist
func (c *Config) Get(property string) string {
	val := c.p[property]
	if val == "" {
		panic("The requested property was not found in the config file: " + property)
	}
	return val
}

// GetInt fetches the property as string and attempts at parsing it as duration string
func (c *Config) GetInt(property string) int {
	sInt := c.Get(property)

	i, err := strconv.Atoi(sInt)
	if err != nil {
		panic(fmt.Sprintf("Error while parsing value of property: %s. Stringvalue: %s, Error: %s", property, sInt, err))
	}
	return i
}

// GetDuration fetches the property as string and attempts at parsing it as duration string
func (c *Config) GetDuration(property string) (time.Duration, error) {
	sDur := c.Get(property)

	dur, err := time.ParseDuration(sDur)
	if err != nil {
		return 0, fmt.Errorf("Error while parsing value of property: %s. Stringvalue: %s, Error: %w", property, sDur, err)
	}
	return dur, nil
}

// CreateConfig creates a new Config object and initializes it with the given config file ref.
func CreateConfig(path string, name string, properties []string) Config {
	var config Config = Config{
		p: make(map[string]string),
	}

	dir, _ := os.Getwd()
	log.Printf("Initializing config.. at path: %s", dir)

	viper.SetConfigName(name)         // name of config file (without extension)
	viper.AddConfigPath(path)         // path to look for the config file during normal exec
	viper.AddConfigPath("../" + path) // path to look for the config file during package test

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Fatal("Error: config file couldn't be found")
		} else {
			// Config file was found but another error was produced
			log.Fatal("Error while reading config file")
		}
		panic(err)
	}

	log.Println("...done!")

	config.LoadProperties(properties)
	return config
}

// LoadProperties attempts to load all provided properties from the config file into memory
func (c *Config) LoadProperties(properites []string) {
	for _, prop := range properites {
		val := viper.GetString(prop)
		c.p[prop] = val
	}
}
