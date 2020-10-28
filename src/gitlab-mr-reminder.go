package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type groupMembers struct {
	Username  string `json:"username"`
	AccessLVL int    `json:"access_level"`
}

type mergeRequests struct {
	Title     string    `json:"title"`
	ID        int       `json:"iid"`
	CreatedAt time.Time `json:"created_at"`
	Author    struct {
		Name string `json:"name"`
	} `json:"author"`
	References struct {
		FullRef string `json:"full"`
	} `json:"references"`
	UserNoteCount int    `json:"user_notes_count"`
	WebUrl        string `json:"web_url"`
}

func (m *mergeRequests) Filter() bool {
	if time.Since(m.CreatedAt).Hours() > 1 {
		if !strings.Contains(m.Title, "WIP") {
			if m.UserNoteCount == 0 {
				return true
			}
		}
	}
	return false
}

func getMembers(token *string, gitlabDomain *string) *[]groupMembers {
	client := &http.Client{}
	r, err := http.NewRequest("GET", fmt.Sprintf("https://%v/api/v4/groups/devops/members?per_page=100", *gitlabDomain), nil)
	r.Header.Set("Private-Token", *token)
	if err != nil {
		log.Fatal(err.Error())
	}
	resp, err := client.Do(r)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err.Error())
	}
	if resp.StatusCode != 200 {
		log.Fatal(resp.Status)
	}
	groupMembers := []groupMembers{}
	jsonErr := json.Unmarshal(body, &groupMembers)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}
	return &groupMembers

}

func getMergeRequests(gitlabToken *string, gitlabDomain *string, groupMember *string) *[]mergeRequests {
	client := &http.Client{}
	r, err := http.NewRequest("GET", fmt.Sprintf("https://%v//api/v4/merge_requests?author_username=%v&state=opened&per_page=100", *gitlabDomain, *groupMember), nil)
	r.Header.Set("Private-Token", *gitlabToken)
	if err != nil {
		log.Fatal(err.Error())
	}
	resp, err := client.Do(r)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err.Error())
	}
	if resp.StatusCode != 200 {
		log.Fatal(resp.Status)
	}
	mergeRequests := []mergeRequests{}
	jsonErr := json.Unmarshal(body, &mergeRequests)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}
	return &mergeRequests

}

func postToSlack(slackWebhook *string, slackChannel *string, msg *[]string) {
	client := &http.Client{}
	var jsonStr = []byte(fmt.Sprintf(`{"channel": "%v", "username": "mr-reminder", "text": "Please can anyone have a look at the folowing MR? They are older than 1h and have no comments: \n%v", "icon_emoji": ":eyes:"}`, *slackChannel, strings.Join(*msg, "\n")))
	log.Println("Posting to slack")
	fmt.Println(string(jsonStr))
	r, err := http.NewRequest("POST", *slackWebhook, bytes.NewBuffer(jsonStr))
	r.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.Fatal(err.Error())
	}
	resp, err := client.Do(r)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Fatal(resp)
	}
}

func main() {
	gitlabDomain := os.Getenv("GITLAB_DOMAIN")
	gitlabToken := os.Getenv("GITLAB_TOKEN")
	slackWebhook := os.Getenv("SLACK_WEBHOOK")
	slackChannel := os.Getenv("SLACK_CHANNEL")
	groupMembers := *getMembers(&gitlabToken, &gitlabDomain)
	var mrList []string
	for _, groupMember := range groupMembers {
		if groupMember.AccessLVL >= 40 {
			fmt.Printf("Checking Merge request for member %v\n", groupMember.Username)
			mergeRequestsPerProject := getMergeRequests(&gitlabToken, &gitlabDomain, &groupMember.Username)
			for _, mergeRequest := range *mergeRequestsPerProject {
				if mergeRequest.Filter() {
					log.Printf("Found: %v", mergeRequest.Title)
					mrList = append(mrList, fmt.Sprintf("[%v][%v] %v %v", mergeRequest.References.FullRef, mergeRequest.Author.Name, mergeRequest.Title, mergeRequest.WebUrl))
				}
			}
		}
	}
	if len(mrList) > 0 {
		postToSlack(&slackWebhook, &slackChannel, &mrList)
	} else {
		fmt.Println("Nothing found, nothing to do")
	}
}
