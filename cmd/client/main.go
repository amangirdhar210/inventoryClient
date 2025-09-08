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
	"text/tabwriter"
	"time"
)

const serverBaseURL = "http://localhost:8080"

type Product struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Quantity int     `json:"quantity"`
}

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
	body, statusCode, err := c.makeRequest("POST", "/products", payload)
	if err != nil {
		return err
	}

	return handleProductResponse(body, statusCode, "Product added successfully.")
}

func (c *Client) getProduct(reader *bufio.Reader) error {
	fmt.Println("\n-> Getting a Product...")
	id := readString(reader, "   Enter Product ID: ")
	body, statusCode, err := c.makeRequest("GET", "/products/"+id, nil)
	if err != nil {
		return err
	}
	return handleProductResponse(body, statusCode, "")
}

func (c *Client) listAllProducts() error {
	fmt.Println("\n-> Listing All Products...")
	body, statusCode, err := c.makeRequest("GET", "/products", nil)
	if err != nil {
		return err
	}

	if statusCode != http.StatusOK {
		return printErrorResponse(body)
	}

	var products []Product
	if err := json.Unmarshal(body, &products); err != nil {
		return fmt.Errorf("failed to decode product list: %w", err)
	}

	fmt.Println("\n<- Server Response:")
	if len(products) == 0 {
		fmt.Println("No products found in inventory.")
	} else {
		printProductsTable(products)
	}
	return nil
}

func (c *Client) sellProduct(reader *bufio.Reader) error {
	fmt.Println("\n-> Selling Product...")
	id := readString(reader, "   Enter Product ID: ")
	quantity := readInt(reader, "   Enter Quantity to Sell: ")

	payload := map[string]interface{}{"quantity": quantity}
	body, statusCode, err := c.makeRequest("POST", fmt.Sprintf("/products/%s/sell", id), payload)
	if err != nil {
		return err
	}

	return handleProductResponse(body, statusCode, "Sale processed successfully.")
}

func (c *Client) restockProduct(reader *bufio.Reader) error {
	fmt.Println("\n-> Restocking a Product...")
	id := readString(reader, "   Enter Product ID: ")
	quantity := readInt(reader, "   Enter Quantity to Restock: ")

	payload := map[string]interface{}{"quantity": quantity}
	body, statusCode, err := c.makeRequest("POST", fmt.Sprintf("/products/%s/restock", id), payload)
	if err != nil {
		return err
	}

	return handleProductResponse(body, statusCode, "Product restocked successfully.")
}

func (c *Client) updateProductPrice(reader *bufio.Reader) error {
	fmt.Println("\n-> Updating Product Price...")
	id := readString(reader, "   Enter Product ID: ")
	price := readFloat(reader, "   Enter New Price: ")

	payload := map[string]interface{}{"price": price}
	body, statusCode, err := c.makeRequest("PUT", fmt.Sprintf("/products/%s/price", id), payload)
	if err != nil {
		return err
	}

	return handleProductResponse(body, statusCode, "Price updated successfully.")
}

func (c *Client) deleteProduct(reader *bufio.Reader) error {
	fmt.Println("\n-> Deleting a Product...")
	id := readString(reader, "   Enter Product ID to Delete: ")
	body, statusCode, err := c.makeRequest("DELETE", "/products/"+id, nil)
	if err != nil {
		return err
	}

	if statusCode == http.StatusNoContent {
		fmt.Println("\n<- Server Response:\nProduct deleted successfully.")
		return nil
	}
	if statusCode != http.StatusOK {
		return printErrorResponse(body)
	}

	return printMessageResponse(body)
}

func (c *Client) getInventoryValue() error {
	fmt.Println("\n-> Getting Total Inventory Value...")
	body, statusCode, err := c.makeRequest("GET", "/inventory/value", nil)
	if err != nil {
		return err
	}

	if statusCode != http.StatusOK {
		return printErrorResponse(body)
	}

	var result struct {
		TotalValue float64 `json:"total_value"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to decode inventory value: %w", err)
	}

	fmt.Println("\n<- Server Response:")
	fmt.Printf("   Total Inventory Value: $%.2f\n", result.TotalValue)
	return nil
}

func (c *Client) makeRequest(method, path string, payload interface{}) ([]byte, int, error) {
	var body io.Reader
	if payload != nil {
		jsonPayload, err := json.Marshal(payload)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal JSON payload: %w", err)
		}
		body = bytes.NewBuffer(jsonPayload)
	}

	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

func printProductsTable(products []Product) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(writer, "ID\tNAME\tPRICE\tQUANTITY")
	fmt.Fprintln(writer, "--\t----\t-----\t--------")
	for _, p := range products {
		fmt.Fprintf(writer, "%s\t%s\t$%.2f\t%d\n",
			p.ID, p.Name, p.Price, p.Quantity)
	}
	writer.Flush()
}

func handleProductResponse(body []byte, statusCode int, successMessage string) error {
	if statusCode != http.StatusOK && statusCode != http.StatusCreated {
		return printErrorResponse(body)
	}
	var product Product
	if err := json.Unmarshal(body, &product); err != nil {
		return fmt.Errorf("failed to decode product: %w", err)
	}

	fmt.Println("\n<- Server Response:")
	if successMessage != "" {
		fmt.Printf("   %s\n", successMessage)
	}
	printProductsTable([]Product{product})
	return nil
}

func printErrorResponse(body []byte) error {
	var errorResponse struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &errorResponse); err == nil && errorResponse.Error != "" {
		fmt.Printf("\n Error from server: %s\n", errorResponse.Error)
		return fmt.Errorf("server returned an error")
	}

	rawError := strings.TrimSpace(string(body))
	if rawError != "" {
		fmt.Printf("\n An unknown error occurred: %s\n", rawError)
	}
	return fmt.Errorf("server returned a non-2xx status code with an unhandled error format")
}

func printMessageResponse(body []byte) error {
	var response struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &response); err == nil && response.Message != "" {
		fmt.Printf("\n<- Server Response:\n%s\n", response.Message)
		return nil
	}
	fmt.Printf("\n<- Server Response:\n%s\n", string(body))
	return nil
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
