package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
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
	WebURL        string `json:"web_url"`
}

func (m *mergeRequests) Filter(mrAge float64) bool {
	if time.Since(m.CreatedAt).Hours() > mrAge {
		if !strings.Contains(m.Title, "WIP") {
			if m.UserNoteCount == 0 {
				return true
			}
		}
	}
	return false
}

func getMembers(token *string, gitlabDomain *string, groupName *string) *[]groupMembers {
	client := &http.Client{}
	r, err := http.NewRequest("GET", fmt.Sprintf("https://%v/api/v4/groups/%v/members?per_page=100", *gitlabDomain, *groupName), nil)
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
	r, err := http.NewRequest("GET", fmt.Sprintf("https://%v//api/v4/merge_requests?author_username=%v&scope=all&state=opened&per_page=100", *gitlabDomain, *groupMember), nil)
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

func postToteams(teamsWebhook *string, teamsChannel *string, msg *[]string) {
	client := &http.Client{}
	var jsonStr = []byte(fmt.Sprintf(`{"channel": "%v", "username": "mr-reminder", "text": "Please can anyone have a look at the folowing MR: \n%v", "icon_emoji": ":eyes:"}`, *teamsChannel, strings.Join(*msg, "\n")))
	log.Println("Posting to teams")
	r, err := http.NewRequest("POST", *teamsWebhook, bytes.NewBuffer(jsonStr))
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
	groupName := os.Getenv("GITLAB_GROUP_NAME")
	groupMemberLvl, _ := strconv.Atoi(os.Getenv("GITLAB_GROUP_MEMBER_LEVEL"))
	mrAge, _ := strconv.ParseFloat(os.Getenv("GITLAB_MR_AGE"), 32)
	teamsWebhook := os.Getenv("TEAMS_WEBHOOK")
	teamsChannel := os.Getenv("TEAMS_CHANNEL")
	if ( groupName == "" || gitlabDomain == "" || gitlabToken == "" || teamsWebhook == "") {
		log.Printf("GITLAB_DOMAIN: %v GITLAB_TOKEN: %v GITLAB_GROUP_NAME: %v TEAMS_WEBHOOK: %v",gitlabDomain,gitlabToken,groupName,teamsWebhook)
		log.Fatal("Missing configuration variables")
	}
	groupMembers := *getMembers(&gitlabToken, &gitlabDomain, &groupName)
	var mrList []string
	for _, groupMember := range groupMembers {
		if groupMember.AccessLVL >= groupMemberLvl {
			log.Printf("Checking Merge request for member %v\n", groupMember.Username)
			mergeRequestsPerProject := getMergeRequests(&gitlabToken, &gitlabDomain, &groupMember.Username)
			for _, mergeRequest := range *mergeRequestsPerProject {
				if mergeRequest.Filter((mrAge)) {
					log.Printf("Found: %v", mergeRequest.Title)
					mrList = append(mrList, fmt.Sprintf("[%v][%v] %v %v", mergeRequest.References.FullRef, mergeRequest.Author.Name, mergeRequest.Title, mergeRequest.WebURL))
				}
			}
		}
	}
	if len(mrList) > 0 {
		postToteams(&teamsWebhook, &teamsChannel, &mrList)
	} else {
		log.Println("Nothing found, nothing to do")
	}
}