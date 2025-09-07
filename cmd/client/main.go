package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const serverBaseURL = "http://localhost:8080"

type Client struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		baseURL:    serverBaseURL,
	}
}

func main() {
	client := NewClient()
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("--- Inventory Management API Client ---")
	fmt.Println("-------------------------------------")

	for {
		displayMenu()
		choice := readString(reader, "Enter your choice: ")

		var err error
		switch choice {
		case "1":
			err = client.addProduct(reader)
		case "2":
			err = client.getProduct(reader)
		case "3":
			err = client.listAllProducts()
		case "4":
			err = client.sellProduct(reader)
		case "5":
			err = client.restockProduct(reader)
		case "6":
			err = client.updateProductPrice(reader)
		case "7":
			err = client.deleteProduct(reader)
		case "8":
			err = client.getInventoryValue()
		case "9":
			fmt.Println("Exiting client.")
			return
		default:
			fmt.Println("Invalid choice. Please select a valid option.")
		}

		if err != nil {
			log.Printf("An error occurred: %v", err)
		}
		fmt.Println("-------------------------------------")
	}
}

func displayMenu() {
	fmt.Println("\nAvailable Commands:")
	fmt.Println("1. Add Product")
	fmt.Println("2. Get Product by ID")
	fmt.Println("3. List All Products")
	fmt.Println("4. Sell Product")
	fmt.Println("5. Restock Product")
	fmt.Println("6. Update Product Price")
	fmt.Println("7. Delete Product")
	fmt.Println("8. Get Total Inventory Value")
	fmt.Println("9. Exit")
}

func (c *Client) addProduct(reader *bufio.Reader) error {
	fmt.Println("\n-> Adding a new Product...")
	name := readString(reader, "   Enter Name: ")
	price := readFloat(reader, "   Enter Price: ")
	quantity := readInt(reader, "   Enter Quantity: ")

	payload := map[string]interface{}{"name": name, "price": price, "quantity": quantity}
	return c.makeRequest("POST", "/products", payload)
}

func (c *Client) getProduct(reader *bufio.Reader) error {
	fmt.Println("\n-> Getting a Product...")
	id := readString(reader, "   Enter Product ID: ")
	return c.makeRequest("GET", "/products/"+id, nil)
}

func (c *Client) listAllProducts() error {
	fmt.Println("\n-> Listing All Products...")
	return c.makeRequest("GET", "/products", nil)
}

func (c *Client) sellProduct(reader *bufio.Reader) error {
	fmt.Println("\n-> Selling Product...")
	id := readString(reader, "   Enter Product ID: ")
	quantity := readInt(reader, "   Enter Quantity to Sell: ")

	payload := map[string]interface{}{"quantity": quantity}
	return c.makeRequest("POST", fmt.Sprintf("/products/%s/sell", id), payload)
}

func (c *Client) restockProduct(reader *bufio.Reader) error {
	fmt.Println("\n-> Restocking a Product...")
	id := readString(reader, "   Enter Product ID: ")
	quantity := readInt(reader, "   Enter Quantity to Restock: ")

	payload := map[string]interface{}{"quantity": quantity}
	return c.makeRequest("POST", fmt.Sprintf("/products/%s/restock", id), payload)
}

func (c *Client) updateProductPrice(reader *bufio.Reader) error {
	fmt.Println("\n-> Updating Product Price...")
	id := readString(reader, "   Enter Product ID: ")
	price := readFloat(reader, "   Enter New Price: ")

	payload := map[string]interface{}{"price": price}
	return c.makeRequest("PUT", fmt.Sprintf("/products/%s/price", id), payload)
}

func (c *Client) deleteProduct(reader *bufio.Reader) error {
	fmt.Println("\n-> Deleting a Product...")
	id := readString(reader, "   Enter Product ID to Delete: ")
	return c.makeRequest("DELETE", "/products/"+id, nil)
}

func (c *Client) getInventoryValue() error {
	fmt.Println("\n-> Getting Total Inventory Value...")
	return c.makeRequest("GET", "/inventory/value", nil)
}

func (c *Client) makeRequest(method, path string, payload interface{}) error {
	var body io.Reader
	if payload != nil {
		jsonPayload, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON payload: %w", err)
		}
		body = bytes.NewBuffer(jsonPayload)
	}

	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("\n<- Server Response (Status: %s)\n", resp.Status)
	return printPrettyResponse(resp.Body)
}

func readString(reader *bufio.Reader, prompt string) string {
	fmt.Print(prompt)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func readInt(reader *bufio.Reader, prompt string) int {
	for {
		inputStr := readString(reader, prompt)
		val, err := strconv.Atoi(inputStr)
		if err == nil {
			return val
		}
		fmt.Println("   Error: Please enter a valid whole number.")
	}
}

func readFloat(reader *bufio.Reader, prompt string) float64 {
	for {
		inputStr := readString(reader, prompt)
		val, err := strconv.ParseFloat(inputStr, 64)
		if err == nil {
			return val
		}
		fmt.Println("   Error: Please enter a valid number (e.g., 49.99).")
	}
}

func printPrettyResponse(body io.Reader) error {
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if len(bodyBytes) == 0 {
		fmt.Println("[Empty Response Body]")
		return nil
	}

	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, bodyBytes, "", "  "); err != nil {
		fmt.Println(string(bodyBytes))
	} else {
		fmt.Println(prettyJSON.String())
	}
	return nil
}
