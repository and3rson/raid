package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
)

const URLPattern = "https://t.me/s/%s"

var MessagesSel = cascadia.MustCompile(".tgme_widget_message")
var AuthorSel = cascadia.MustCompile(".tgme_widget_message_author span")
var TextSel = cascadia.MustCompile(".tgme_widget_message_text")
var DateSel = cascadia.MustCompile(".tgme_widget_message_footer time[datetime]")

type ChannelClient struct {
	client  *http.Client
	channel string
}

type Message struct {
	ID int64
	Author string
	Text []string
	Date time.Time
}

func (m Message) String() string {
	return fmt.Sprintf(
		"[ID:%d Author:%s Text:%s Date:%s]",
		m.ID, m.Author, m.Text, m.Date,
	)
}

func NewChannelClient(channel string) *ChannelClient {
	return &ChannelClient{
		&http.Client{},
		channel,
	}
}

func getText(node *html.Node) string {
	return strings.Join(getLines(node), " ")
}

func getLines(node *html.Node) []string {
	parts := []string{}
	if node.Type == html.TextNode {
		text := strings.TrimSpace(node.Data)
		if len(text) > 0 {
			parts = append(parts, text)
		}
	}
	for n := node.FirstChild; n != nil; n = n.NextSibling {
		parts = append(parts, getLines(n)...)
	}
	return parts
}

func getAttr(node *html.Node, key string) string {
	for _, attr := range node.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func (c *ChannelClient) FetchMessages(before int64) ([]Message, error) {
	messages := []Message{}
	url := fmt.Sprintf(URLPattern, c.channel)
	if before != 0 {
		url = fmt.Sprintf("%s?before=%d", url, before)
	}
	req, err := http.NewRequest("POST", url, nil)
	req.Header.Add("X-Requested-With", "XMLHttpRequest")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(resp.Body)
	var data string
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}
	root, err := html.Parse(bytes.NewReader([]byte(data)))
	if err != nil {
		return nil, err
	}
	nodes := MessagesSel.MatchAll(root)
	for _, node := range nodes {
		authorNode := AuthorSel.MatchFirst(node)
		textNode := TextSel.MatchFirst(node)
		dateNode := DateSel.MatchFirst(node)
		dateTimeNode := getAttr(dateNode, "datetime")
		dataPost := getAttr(node, "data-post")
		parts := strings.Split(dataPost, "/")
		id, err := strconv.ParseInt(parts[len(parts) - 1], 10, 64)
		if err != nil {
			return nil, err
		}
		datetime, err := time.Parse(time.RFC3339, dateTimeNode)
		if err != nil {
			return nil, err
		}
		datetime = datetime.In(Timezone)
		messages = append(messages, Message{id, getText(authorNode), getLines(textNode), datetime})
	}
	return messages, nil
}

func (c *ChannelClient) FetchLast(count int) ([]Message, error) {
	messages := []Message{}
	var oldestID int64 = 0
	for len(messages) < count {
		prevMessages, err := c.FetchMessages(oldestID)
		if err != nil {
			return nil, err
		}
		if len(prevMessages) == 0 {
			return messages, nil
		}
		messages = append(prevMessages, messages...)
		oldestID = prevMessages[0].ID
	}
	return messages, nil
}

func (c *ChannelClient) FetchNewer(after int64) ([]Message, error) {
	messages := []Message{}
	var oldestID int64 = 0
	for len(messages) == 0 || messages[0].ID > after {
		prevMessages, err := c.FetchMessages(oldestID)
		if err != nil {
			return nil, err
		}
		if len(prevMessages) == 0 {
			return messages, nil
		}
		messages = append(prevMessages, messages...)
		oldestID = prevMessages[0].ID
	}
	result := []Message{}
	for _, message := range messages {
		if message.ID > after {
			result = append(result, message)
		}
	}
	return result, nil
}
