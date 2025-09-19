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
	token      string
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

	for {
		if client.token == "" {
			client.runLoggedOutLoop(reader)
		} else {
			client.runLoggedInLoop(reader)
		}
	}
}

func (c *Client) runLoggedOutLoop(reader *bufio.Reader) {
	for {
		fmt.Println("\n-------------------------------------")
		fmt.Println("You are not logged in.")
		fmt.Println("1. Login")
		fmt.Println("2. Exit")
		choice := readString(reader, "Enter your choice: ")

		switch choice {
		case "1":
			err := c.login(reader)
			if err != nil {
				log.Printf("Login failed: %v", err)
			} else {
				fmt.Println("\nLogin successful!")
				return
			}
		case "2":
			fmt.Println("Exiting client.")
			os.Exit(0)
		default:
			fmt.Println("Invalid choice. Please select a valid option.")
		}
	}
}

func (c *Client) runLoggedInLoop(reader *bufio.Reader) {
	fmt.Println("\n-------------------------------------")
	fmt.Println("You are logged in.")

	for {
		displayLoggedInMenu()
		choice := readString(reader, "Enter your choice: ")
		var err error

		switch choice {
		case "1":
			err = c.addProduct(reader)
		case "2":
			err = c.getProduct(reader)
		case "3":
			err = c.listAllProducts()
		case "4":
			err = c.sellProduct(reader)
		case "5":
			err = c.restockProduct(reader)
		case "6":
			err = c.updateProductPrice(reader)
		case "7":
			err = c.deleteProduct(reader)
		case "8":
			err = c.getInventoryValue()
		case "9":
			c.logout()
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

func displayLoggedInMenu() {
	fmt.Println("\nAvailable Commands:")
	fmt.Println("1. Add Product")
	fmt.Println("2. Get Product by ID")
	fmt.Println("3. List All Products")
	fmt.Println("4. Sell Product")
	fmt.Println("5. Restock Product")
	fmt.Println("6. Update Product Price")
	fmt.Println("7. Delete Product")
	fmt.Println("8. Get Total Inventory Value")
	fmt.Println("9. Logout")
}

func (c *Client) login(reader *bufio.Reader) error {
	fmt.Println("\n-> Logging in...")
	email := readString(reader, "   Enter Email: ")
	password := readString(reader, "   Enter Password: ")

	payload := map[string]string{"email": email, "password": password}
	body, statusCode, err := c.makeRequest("POST", "/login", payload)
	if err != nil {
		return err
	}

	if statusCode != http.StatusOK {
		return printErrorResponse(body)
	}

	var response struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to decode login response: %w", err)
	}

	c.token = response.Token
	return nil
}

func (c *Client) logout() {
	c.token = ""
	fmt.Println("\nYou have been logged out.")
}

func (c *Client) makeRequest(method, path string, payload any) ([]byte, int, error) {
	var body io.Reader
	if payload != nil {
		jsonPayload, err := json.Marshal(payload)
		if err != nil {
			return nil, 0, err
		}
		body = bytes.NewBuffer(jsonPayload)
	}

	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return nil, 0, err
	}

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	return respBody, resp.StatusCode, nil
}

func (c *Client) addProduct(reader *bufio.Reader) error {
	fmt.Println("\n-> Adding a new Product...")
	name := readString(reader, "   Enter Name: ")
	price := readFloat(reader, "   Enter Price: ")
	quantity := readInt(reader, "   Enter Quantity: ")

	payload := map[string]any{"name": name, "price": price, "quantity": quantity}
	body, statusCode, err := c.makeRequest("POST", "/api/products", payload)
	if err != nil {
		return err
	}

	return handleProductResponse(body, statusCode, "Product added successfully.")
}

func (c *Client) getProduct(reader *bufio.Reader) error {
	fmt.Println("\n-> Getting a Product...")
	id := readString(reader, "   Enter Product ID: ")
	body, statusCode, err := c.makeRequest("GET", "/api/products/"+id, nil)
	if err != nil {
		return err
	}
	return handleProductResponse(body, statusCode, "")
}

func (c *Client) listAllProducts() error {
	fmt.Println("\n-> Listing All Products...")
	body, statusCode, err := c.makeRequest("GET", "/api/products", nil)
	if err != nil {
		return err
	}

	if statusCode != http.StatusOK {
		return printErrorResponse(body)
	}

	var products []Product
	if err := json.Unmarshal(body, &products); err != nil {
		return err
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

	payload := map[string]any{"quantity": quantity}
	body, statusCode, err := c.makeRequest("PATCH", fmt.Sprintf("/api/products/%s/sell", id), payload)
	if err != nil {
		return err
	}

	return handleProductResponse(body, statusCode, "Sale processed successfully.")
}

func (c *Client) restockProduct(reader *bufio.Reader) error {
	fmt.Println("\n-> Restocking a Product...")
	id := readString(reader, "   Enter Product ID: ")
	quantity := readInt(reader, "   Enter Quantity to Restock: ")

	payload := map[string]any{"quantity": quantity}
	body, statusCode, err := c.makeRequest("PATCH", fmt.Sprintf("/api/products/%s/restock", id), payload)
	if err != nil {
		return err
	}
	return handleProductResponse(body, statusCode, "Product restocked successfully.")
}

func (c *Client) updateProductPrice(reader *bufio.Reader) error {
	fmt.Println("\n-> Updating Product Price...")
	id := readString(reader, "   Enter Product ID: ")
	price := readFloat(reader, "   Enter New Price: ")

	payload := map[string]any{"price": price}
	body, statusCode, err := c.makeRequest("PATCH", fmt.Sprintf("/api/products/%s/price", id), payload)
	if err != nil {
		return err
	}

	return handleMessageResponse(body, statusCode, "Price updated successfully.")
}

func (c *Client) deleteProduct(reader *bufio.Reader) error {
	fmt.Println("\n-> Deleting a Product...")
	id := readString(reader, "   Enter Product ID to Delete: ")
	body, statusCode, err := c.makeRequest("DELETE", "/api/products/"+id, nil)
	if err != nil {
		return err
	}
	return handleMessageResponse(body, statusCode, "Product deleted successfully.")
}

func (c *Client) getInventoryValue() error {
	fmt.Println("\n-> Getting Total Inventory Value...")
	body, statusCode, err := c.makeRequest("GET", "/api/inventory/value", nil)
	if err != nil {
		return err
	}

	if statusCode != http.StatusOK {
		return printErrorResponse(body)
	}

	var result struct {
		Value float64 `json:"inventory_value"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	fmt.Println("\n<- Server Response:")
	fmt.Printf("   Total Inventory Value: $%.2f\n", result.Value)
	return nil
}

func printProductsTable(products []Product) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(writer, "ID\tNAME\tPRICE\tQUANTITY")
	fmt.Fprintln(writer, "--\t----\t-----\t--------")
	for _, p := range products {
		fmt.Fprintf(writer, "%s\t%s\t$%.2f\t%d\n", p.ID, p.Name, p.Price, p.Quantity)
	}
	writer.Flush()
}

func handleProductResponse(body []byte, statusCode int, successMessage string) error {
	if statusCode != http.StatusOK && statusCode != http.StatusCreated {
		return printErrorResponse(body)
	}
	var product Product
	if err := json.Unmarshal(body, &product); err != nil {
		return err
	}

	fmt.Println("\n<- Server Response:")
	if successMessage != "" {
		fmt.Printf("   %s\n", successMessage)
	}
	return nil
}

func handleMessageResponse(body []byte, statusCode int, defaultMessage string) error {
	if statusCode != http.StatusOK {
		return printErrorResponse(body)
	}
	var response struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &response); err == nil && response.Message != "" {
		fmt.Printf("\n<- Server Response:\n   %s\n", response.Message)
	} else {
		fmt.Printf("\n<- Server Response:\n   %s\n", defaultMessage)
	}
	return nil
}

func printErrorResponse(body []byte) error {
	var errResp struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != "" {
		return fmt.Errorf("server error: %s", errResp.Error)
	}
	return fmt.Errorf("unknown server error: %s", string(body))
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
