package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

const (
	baseURL = "https://dev.azure.com"
	patEnv  = "AZURE_DEVOPS_PAT" // Environment variable for storing PAT
)

type Pipeline struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type PipelinesResponse struct {
	Count     int        `json:"count"`
	Pipelines []Pipeline `json:"value"`
}

func getPipelines(organization, project, pat string) ([]Pipeline, error) {
	url := fmt.Sprintf("%s/%s/%s/_apis/pipelines?api-version=7.0", baseURL, organization, project)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add PAT as Basic Auth header
	req.SetBasicAuth("", pat)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch pipelines, status: %s", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var pipelinesResponse PipelinesResponse
	if err := json.Unmarshal(body, &pipelinesResponse); err != nil {
		return nil, err
	}

	return pipelinesResponse.Pipelines, nil
}

func promptUser(prompt string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func persistPATToShell(pat string) error {
	// Determine the user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	// Determine which shell RC file to write to
	rcFile := fmt.Sprintf("%s/.bashrc", homeDir) // Change to .zshrc for zsh users
	if shell := os.Getenv("SHELL"); strings.Contains(shell, "zsh") {
		rcFile = fmt.Sprintf("%s/.zshrc", homeDir)
	}

	// Check if the file already contains the PAT
	existingLines, err := ioutil.ReadFile(rcFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read shell config file: %v", err)
	}

	if strings.Contains(string(existingLines), fmt.Sprintf("export %s=", patEnv)) {
		return nil // PAT already exists
	}

	// Append the PAT to the shell RC file
	f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open shell config file: %v", err)
	}
	defer f.Close()

	_, err = f.WriteString(fmt.Sprintf("\n# Added by Azure DevOps PAT setup\nexport %s=%s\n", patEnv, pat))
	if err != nil {
		return fmt.Errorf("failed to write to shell config file: %v", err)
	}

	fmt.Printf("PAT saved to %s. Restart your terminal or run `source %s` to apply the changes.\n", rcFile, rcFile)
	return nil
}

func main() {
	// Prompt for inputs interactively
	organization := promptUser("Enter your Azure DevOps organization: ")
	project := promptUser("Enter your Azure DevOps project: ")

	// Check if PAT exists in the environment
	pat := os.Getenv(patEnv)
	if pat == "" {
		// Prompt the user for PAT if not already set
		pat = promptUser("Enter your Azure DevOps PAT: ")

		// Persist the PAT in the shell configuration file
		if err := persistPATToShell(pat); err != nil {
			log.Fatalf("Error saving PAT to shell: %v", err)
		}
	}

	// Validate inputs
	if organization == "" || project == "" || pat == "" {
		log.Fatal("All inputs (organization, project, PAT) are required.")
	}

	// Fetch pipelines
	pipelines, err := getPipelines(organization, project, pat)
	if err != nil {
		log.Fatalf("Error fetching pipelines: %v", err)
	}

	// Display pipelines
	fmt.Println("Azure DevOps Pipelines:")
	for _, pipeline := range pipelines {
		fmt.Printf("ID: %d, Name: %s\n", pipeline.ID, pipeline.Name)
	}
}

