//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	fmt.Println("Fixing protobuf negative index errors in generated files...")
	
	// Path to the module in go.mod cache
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("Error getting user home directory: %v\n", err)
			return
		}
		gopath = filepath.Join(home, "go")
	}
	
	// Find all versions of the module
	modPath := filepath.Join(gopath, "pkg", "mod", "github.com", "shiestapoi", "whatsmeow@*")
	dirs, err := filepath.Glob(modPath)
	if err != nil {
		fmt.Printf("Error finding module: %v\n", err)
		return
	}
	
	if len(dirs) == 0 {
		fmt.Println("No module directories found. Make sure the module is installed.")
		return
	}
	
	// Process each version
	for _, dir := range dirs {
		fmt.Printf("Processing module directory: %s\n", dir)
		
		// Find all .pb.go files in the proto directory
		err = filepath.Walk(filepath.Join(dir, "proto"), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			
			if !info.IsDir() && strings.HasSuffix(path, ".pb.go") {
				// Fix the file
				if err := fixFile(path); err != nil {
					fmt.Printf("Error fixing %s: %v\n", path, err)
				} else {
					fmt.Printf("Fixed %s\n", path)
				}
			}
			
			return nil
		})
		
		if err != nil {
			fmt.Printf("Error walking directory %s: %v\n", dir, err)
		}
	}
	
	fmt.Println("All protobuf files have been patched.")
	fmt.Println("If you're still experiencing issues, please report them at https://github.com/shiestapoi/whatsmeow/issues")
}

func fixFile(path string) error {
	// Make the file writable if needed
	err := os.Chmod(path, 0644)
	if err != nil {
		return fmt.Errorf("could not make file writable: %v", err)
	}
	
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	
	// Replace all patterns of array[-1] with array[0]
	patterns := []string{
		`x\[-1\]`,
		`dv\[-1\]`,
		`sv\[-1\]`,
		`bv\[-1\]`,
	}
	
	modifiedContent := string(content)
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		replacement := strings.Replace(pattern, `\[-1\]`, "[0]", 1)
		replacement = strings.Replace(replacement, `\`, "", -1)
		modifiedContent = re.ReplaceAllString(modifiedContent, replacement)
	}
	
	// Write the modified content back
	return ioutil.WriteFile(path, []byte(modifiedContent), 0644)
}
