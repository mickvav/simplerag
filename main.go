package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"
	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"
	"github.com/spf13/cobra"
)

const (
	dbConn     = "postgres://user:password@localhost/dbname?sslmode=disable"
	openaiKey  = "YOUR_OPENAI_API_KEY"
	modelEmbed = "text-embedding-ada-002"
	modelChat  = "gpt-4-turbo"
)

func generateEmbedding(text string) ([]float32, error) {
	client := openai.NewClient(
		option.WithAPIKey(openaiKey),
		option.WithBaseURL("http://127.0.0.1:8012"),
	)
	ctx := context.Background()

	resp, err := client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model: modelEmbed,
		Input: openai.EmbeddingNewParamsInputUnion{
			OfString: param.Opt[string]{Value: text},
		},
		EncodingFormat: "float",
	})
	if err != nil {
		return nil, err
	}
	print(resp.JSON.Data.Raw())
	return []float32{}, nil
}

func storeEmbedding(db *sql.DB, content string, embedding []float32) error {
	embeddingJSON, _ := json.Marshal(embedding)
	_, err := db.Exec("INSERT INTO documents (content, embedding) VALUES ($1, $2)", content, string(embeddingJSON))
	return err
}

func removeDocument(db *sql.DB, content string) error {
	_, err := db.Exec("DELETE FROM documents WHERE content = $1", content)
	return err
}

func queryNearestDocuments(db *sql.DB, embedding []float32, k int) ([]string, error) {
	embeddingJSON, _ := json.Marshal(embedding)
	query := fmt.Sprintf(`
		SELECT content FROM documents
		ORDER BY embedding <-> '%s'::vector
		LIMIT %d`, string(embeddingJSON), k)

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []string
	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			return nil, err
		}
		results = append(results, content)
	}
	return results, nil
}

func generateLLMResponse(prompt string) (string, error) {
	client := openai.NewClient(
		option.WithAPIKey(openaiKey),
		option.WithBaseURL("http://127.0.0.1:8012"),
	)
	ctx := context.Background()

	resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			{
				OfSystem: &openai.ChatCompletionSystemMessageParam{
					Content: openai.ChatCompletionSystemMessageParamContentUnion{
						OfString: param.Opt[string]{
							Value: "You are a helpful assistant.",
						},
					},
				},
			},
			{
				OfSystem: &openai.ChatCompletionSystemMessageParam{
					Content: openai.ChatCompletionSystemMessageParamContentUnion{
						OfString: param.Opt[string]{
							Value:"Use the provided context to answer the user's query.",
						},
					},
				},
			},
			{
				OfUser: &openai.ChatCompletionUserMessageParam{
					Content: openai.ChatCompletionUserMessageParamContentUnion{
						OfString: param.Opt[string]{
							Value: prompt,
						},
					},
				},
			},
		},
	})
	if err != nil {
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}

func main() {
	var filePath string
	var numResults int

	var rootCmd = &cobra.Command{Use: "helper"}

	var addCmd = &cobra.Command{
		Use:   "add -f <file_path>",
		Short: "Add a document to the index",
		Run: func(cmd *cobra.Command, args []string) {
			if filePath == "" {
				log.Fatal("File path is required. Use -f <file_path>")
			}

			data, err := os.ReadFile(filePath)
			if err != nil {
				log.Fatalf("Failed to read file: %v", err)
			}
			content := string(data)

			db, err := sql.Open("postgres", dbConn)
			if err != nil {
				log.Fatal(err)
			}
			defer db.Close()

			embedding, err := generateEmbedding(content)
			if err != nil {
				log.Fatalf("Failed to generate embedding: %v", err)
			}

			err = storeEmbedding(db, content, embedding)
			if err != nil {
				log.Fatalf("Failed to store document: %v", err)
			}

			fmt.Println("Document indexed successfully.")
		},
	}
	addCmd.Flags().StringVarP(&filePath, "file", "f", "", "Path to the document")
	rootCmd.AddCommand(addCmd)

	var removeCmd = &cobra.Command{
		Use:   "remove -f <file_path>",
		Short: "Remove a document from the index",
		Run: func(cmd *cobra.Command, args []string) {
			if filePath == "" {
				log.Fatal("File path is required. Use -f <file_path>")
			}

			data, err := os.ReadFile(filePath)
			if err != nil {
				log.Fatalf("Failed to read file: %v", err)
			}
			content := string(data)

			db, err := sql.Open("postgres", dbConn)
			if err != nil {
				log.Fatal(err)
			}
			defer db.Close()

			err = removeDocument(db, content)
			if err != nil {
				log.Fatalf("Failed to remove document: %v", err)
			}

			fmt.Println("Document removed successfully.")
		},
	}
	removeCmd.Flags().StringVarP(&filePath, "file", "f", "", "Path to the document")
	rootCmd.AddCommand(removeCmd)

	var findCmd = &cobra.Command{
		Use:   "find <query>",
		Short: "Find the most relevant documents",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				log.Fatal("Query is required.")
			}
			queryText := strings.Join(args, " ")

			db, err := sql.Open("postgres", dbConn)
			if err != nil {
				log.Fatal(err)
			}
			defer db.Close()

			embedding, err := generateEmbedding(queryText)
			if err != nil {
				log.Fatalf("Failed to generate embedding: %v", err)
			}

			retrievedDocs, err := queryNearestDocuments(db, embedding, numResults)
			if err != nil {
				log.Fatalf("Failed to retrieve documents: %v", err)
			}

			fmt.Println("Relevant Documents:")
			for _, doc := range retrievedDocs {
				fmt.Println("- " + doc)
			}
		},
	}
	findCmd.Flags().IntVarP(&numResults, "num", "n", 3, "Number of results to return")
	rootCmd.AddCommand(findCmd)

	var helpCmd = &cobra.Command{
		Use:   "help <query>",
		Short: "Run LLM with relevant documents as context",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				log.Fatal("Query is required.")
			}
			queryText := strings.Join(args, " ")

			db, err := sql.Open("postgres", dbConn)
			if err != nil {
				log.Fatal(err)
			}
			defer db.Close()

			embedding, err := generateEmbedding(queryText)
			if err != nil {
				log.Fatal(err)
			}

			retrievedDocs, err := queryNearestDocuments(db, embedding, numResults)
			if err != nil {
				log.Fatal(err)
			}

			context := strings.Join(retrievedDocs, "\n")
			llmPrompt := fmt.Sprintf("Context: %s\n\nQuestion: %s", context, queryText)

			response, err := generateLLMResponse(llmPrompt)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println("AI Response:\n", response)
		},
	}
	helpCmd.Flags().IntVarP(&numResults, "num", "n", 3, "Number of context documents")
	rootCmd.AddCommand(helpCmd)

	rootCmd.Execute()
}
