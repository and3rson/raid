package raid

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/andybalholm/cascadia"
	log "github.com/sirupsen/logrus"
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
	ID     int64
	Author string
	Text   []string
	Date   time.Time
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

func (c *ChannelClient) fetchAndParse(req *http.Request) (*html.Node, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("telegram: post request: %w", err)
	}

	var data string

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("telegram: decode response: %w", err)
	}

	root, err := html.Parse(bytes.NewReader([]byte(data)))
	if err != nil {
		return nil, fmt.Errorf("telegram: parse response as HTML: %w", err)
	}

	return root, nil
}

func (c *ChannelClient) FetchMessages(ctx context.Context, before int64) ([]Message, error) {
	messages := []Message{}
	url := fmt.Sprintf(URLPattern, c.channel)

	if before != 0 {
		url = fmt.Sprintf("%s?before=%d", url, before)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("telegram: create request: %w", err)
	}

	req.Header.Add("X-Requested-With", "XMLHttpRequest")

	// log.Debugf("POST %s", req.URL)

	attempts := 5

	var root *html.Node

	for {
		if root, err = c.fetchAndParse(req); err == nil {
			break
		}

		if errors.Is(err, context.Canceled) {
			return nil, err
		}

		log.Warnf("%v, will retry after 10s", err)
		<-time.After(time.Second * 10)

		attempts--
		if attempts == 0 {
			return nil, err
		}
	}

	nodes := MessagesSel.MatchAll(root)
	for _, node := range nodes {
		authorNode := AuthorSel.MatchFirst(node)
		textNode := TextSel.MatchFirst(node)
		dateNode := DateSel.MatchFirst(node)
		dateTimeNode := getAttr(dateNode, "datetime")
		dataPost := getAttr(node, "data-post")
		parts := strings.Split(dataPost, "/")

		id, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("telegram: parse message ID: %w", err)
		}

		datetime, err := time.Parse(time.RFC3339, dateTimeNode)
		if err != nil {
			return nil, fmt.Errorf("telegram: parse message time: %w", err)
		}
		// Note: datetime is in UTC without timezone here

		messages = append(messages, Message{id, getText(authorNode), getLines(textNode), datetime})
	}

	return messages, nil
}

func (c *ChannelClient) FetchLast(ctx context.Context, count int) ([]Message, error) {
	messages := []Message{}

	var oldestID int64

	for len(messages) < count {
		prevMessages, err := c.FetchMessages(ctx, oldestID)
		if err != nil {
			return nil, err
		}

		if len(prevMessages) == 0 {
			return messages, nil
		}

		messages = append(prevMessages, messages...)

		oldestID = prevMessages[0].ID

		<-time.After(time.Millisecond * 50)
	}

	return messages, nil
}

func (c *ChannelClient) FetchNewer(ctx context.Context, after int64) ([]Message, error) {
	messages := []Message{}

	var oldestID int64

	for len(messages) == 0 || messages[0].ID > after {
		prevMessages, err := c.FetchMessages(ctx, oldestID)
		if err != nil {
			return nil, err
		}

		if len(prevMessages) == 0 {
			return messages, nil
		}

		messages = append(prevMessages, messages...)
		oldestID = prevMessages[0].ID

		<-time.After(time.Millisecond * 50)
	}

	result := []Message{}

	for _, message := range messages {
		if message.ID > after {
			result = append(result, message)
		}
	}

	return result, nil
}
