package utils

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

type TripayClient struct {
	APIKey       string
	PrivateKey   string
	MerchantCode string
	BaseURL      string
}

func NewTripayClient() *TripayClient {
	mode := os.Getenv("TRIPAY_MODE")
	baseURL := "https://tripay.co.id/api-sandbox"
	if mode == "production" {
		baseURL = "https://tripay.co.id/api"
	}

	return &TripayClient{
		APIKey:       os.Getenv("TRIPAY_API_KEY"),
		PrivateKey:   os.Getenv("TRIPAY_PRIVATE_KEY"),
		MerchantCode: os.Getenv("TRIPAY_MERCHANT_CODE"),
		BaseURL:      baseURL,
	}
}

func (t *TripayClient) GenerateSignature(merchantRef string, amount int) string {
	data := t.MerchantCode + merchantRef + fmt.Sprintf("%d", amount)
	h := hmac.New(sha256.New, []byte(t.PrivateKey))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func (t *TripayClient) VerifyCallbackSignature(body []byte, signature string) bool {
	h := hmac.New(sha256.New, []byte(t.PrivateKey))
	h.Write(body)
	expectedSignature := hex.EncodeToString(h.Sum(nil))
	return expectedSignature == signature
}

func (t *TripayClient) GetPaymentChannels() ([]interface{}, error) {
	url := fmt.Sprintf("%s/merchant/payment-channel", t.BaseURL)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+t.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Success bool          `json:"success"`
		Message string        `json:"message"`
		Data    []interface{} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, errors.New(result.Message)
	}

	return result.Data, nil
}

func (t *TripayClient) CreateTransaction(payload interface{}) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/transaction/create", t.BaseURL)
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+t.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result struct {
		Success bool                   `json:"success"`
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, errors.New(result.Message)
	}

	return result.Data, nil
}


func (t *TripayClient) GetTransactionDetail(reference string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/transaction/detail?reference=%s", t.BaseURL, reference)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+t.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Success bool                   `json:"success"`
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, errors.New(result.Message)
	}

	return result.Data, nil
}

// Helpers
func StringValue(s *string, defaultValue string) string {
	if s == nil {
		return defaultValue
	}
	return *s
}

func StringPtr(s string) *string {
	return &s
}

func InterfaceToStringPtr(i interface{}) *string {
	if i == nil {
		return nil
	}
	s := fmt.Sprintf("%v", i)
	return &s
}
