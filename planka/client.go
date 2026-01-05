package planka

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	BaseURL  string
	Username string
	Password string
	Token    string
	Client   *http.Client
}

type Card struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	ListID  string `json:"listId"`
	BoardID string `json:"boardId"`
}

type CardResponse struct {
	Item Card `json:"item"`
}

type BoardResponse struct {
	Included struct {
		Cards []Card `json:"cards"`
	} `json:"included"`
}

func NewClient(baseURL, username, password string) *Client {
	return &Client{
		BaseURL:  baseURL,
		Username: username,
		Password: password,
		Client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) Login() error {
	url := fmt.Sprintf("%s/access-tokens", c.BaseURL)
	payload := map[string]string{
		"emailOrUsername": c.Username,
		"password":        c.Password,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := c.Client.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed with status: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	token, ok := result["item"].(string)
	if !ok {
		return fmt.Errorf("token not found in response")
	}

	c.Token = token
	return nil
}

func (c *Client) GetCards(boardID, listID string) ([]Card, error) {
	url := fmt.Sprintf("%s/boards/%s?includedFields=cards", c.BaseURL, boardID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch board cards: %d", resp.StatusCode)
	}

	var result BoardResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var filteredCards []Card
	for _, card := range result.Included.Cards {
		if card.ListID == listID {
			filteredCards = append(filteredCards, card)
		}
	}

	return filteredCards, nil
}

func (c *Client) CreateCard(boardID, listID, name string) error {
	url := fmt.Sprintf("%s/lists/%s/cards", c.BaseURL, listID)

	// Create a new card at the top (position logic might be needed, but let's assume default works)
	// Planka usually needs ListID and BoardID.
	payload := map[string]interface{}{
		"boardId":  boardID,
		"listId":   listID,
		"name":     name,
		"position": 0,         // Top?
		"type":     "project", // Whitelisted type: project, story
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		return fmt.Errorf("create card failed with status: %d, body: %s", resp.StatusCode, buf.String())
	}
	return nil
}

func (c *Client) DeleteCard(cardID string) error {
	url := fmt.Sprintf("%s/cards/%s", c.BaseURL, cardID)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete card failed with status: %d", resp.StatusCode)
	}
	return nil
}
