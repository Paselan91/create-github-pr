package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/google/go-github/v32/github"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
)

func main() {
	ctx := context.Background()

	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
	token := os.Getenv("GITHUB_ACCESS_TOKEN")
	if token == "" {
		log.Fatal("GitHub token not set. Set token environment variable.")
		return
	}
	selectableRepositoriesJson := os.Getenv("SELECTABLE_REPOSITORIES")
	selectableBranchesJson := os.Getenv("SELECTABLE_BRANCHES")
	var selectableRepositories, selectableBranches []string
	if err := json.Unmarshal([]byte(selectableRepositoriesJson), &selectableRepositories); err != nil {
		log.Fatal("Error parsing selectableRepositories:", err)
		return
	}
	if err := json.Unmarshal([]byte(selectableBranchesJson), &selectableBranches); err != nil {
		log.Fatal("Error parsing selectableBranches:", err)
		return
	}

	repositoryOwner := os.Getenv("REPOSITORY_OWNER")
	if token == "" {
		log.Fatal("Repository owner not set. Set token environment variable.")
		return
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)
	reader := bufio.NewReader(os.Stdin)

	// select repositories
	selectedRepositories := []string{}
	surveyMultiSelect("Choose repositories:", selectableRepositories, &selectedRepositories)

	// select base branch
	baseBranch := ""
	surveySelect("Choose a base branch:", selectableBranches, &baseBranch)

	// select compare branch
	compareBranch := ""
	surveySelect("Choose a compare branch:", selectableBranches, &compareBranch)

	// input PR title
	fmt.Print("Enter the PR title: ")
	prTitle, _ := reader.ReadString('\n')
	prTitle = strings.TrimSpace(prTitle)

	// confirm user input
	if confirm := confirmUserInput(selectedRepositories, baseBranch, compareBranch, prTitle, reader); !confirm {
		fmt.Println("Operation cancelled by the user.")
		return
	}

	// create PRs
	for _, repo := range selectedRepositories {
		createPR(ctx, client, repositoryOwner, repo, baseBranch, compareBranch, prTitle)
	}
}

func createPR(ctx context.Context, client *github.Client, repositoryOwner, repo, baseBranch, compareBranch, prTitle string) {
	newPR := &github.NewPullRequest{
		Title: &prTitle,
		Head:  &compareBranch,
		Base:  &baseBranch,
		// Body:   github.String("Automatically generated PR"),
		MaintainerCanModify: github.Bool(true),
	}

	pr, _, err := client.PullRequests.Create(ctx, repositoryOwner, repo, newPR)
	if err != nil {
		fmt.Printf("Failed to create PR for repository %s: %s\n", repo, err)
		return
	}

	fmt.Printf("PR created: %s\n", pr.GetHTMLURL())
}

func surveyMultiSelect(message string, options []string, response *[]string) {
	prompt := &survey.MultiSelect{
		Message: message,
		Options: options,
	}
	err := survey.AskOne(prompt, response)
	if err != nil {
		log.Fatal("Error: occurred", err)
		return
	}
}

func surveySelect(message string, options []string, response *string) {
	prompt := &survey.Select{
		Message: message,
		Options: options,
	}
	err := survey.AskOne(prompt, response)
	if err != nil {
		log.Fatal("Error: occurred", err)
		return
	}
}

func confirmUserInput(selectedRepositories []string, baseBranch, compareBranch, prTitle string, reader *bufio.Reader) bool {
	err := validateInput(selectedRepositories, baseBranch, compareBranch, prTitle)
	if err != nil {
		log.Fatal(err)
		return false
	}

	printUserInput(selectedRepositories, baseBranch, compareBranch, prTitle)

	fmt.Println("Is this information correct? (yes/no)")
	confirm, _ := reader.ReadString('\n')
	return strings.TrimSpace(strings.ToLower(confirm)) == "yes"
}

func validateInput(selectedRepositories []string, baseBranch, compareBranch, prTitle string) error {
	if len(selectedRepositories) == 0 {
		return fmt.Errorf("no repositories selected")
	}
	if baseBranch == "" {
		return fmt.Errorf("base branch not selected")
	}
	if compareBranch == "" {
		return fmt.Errorf("compare branch not selected")
	}
	if prTitle == "" {
		fmt.Println("Warning: PR title is empty")
	}
	return nil
}

func printUserInput(selectedRepositories []string, baseBranch, compareBranch, prTitle string) {
	fmt.Println("---------------------------------")
	fmt.Println("You have entered the following information:")
	fmt.Println("- Target Repositories:")
	for i, repo := range selectedRepositories {
		fmt.Printf("  %d. %s\n", i+1, repo)
	}
	fmt.Printf("- Base Branch:    %s\n", baseBranch)
	fmt.Printf("- Compare Branch: %s\n", compareBranch)
	fmt.Printf("- PR Title:       %s\n", prTitle)
	fmt.Println("---------------------------------")
}
