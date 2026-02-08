package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	joonix "github.com/joonix/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"monorepo/bin-rag-manager/internal/config"
	"monorepo/bin-rag-manager/pkg/embedder"
	"monorepo/bin-rag-manager/pkg/generator"
	"monorepo/bin-rag-manager/pkg/listenhandler"
	"monorepo/bin-rag-manager/pkg/raghandler"
	"monorepo/bin-rag-manager/pkg/retriever"
	"monorepo/bin-rag-manager/pkg/store"
)

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

var rootCmd = &cobra.Command{
	Use:   "rag-manager",
	Short: "RAG Manager Service",
	Long:  `RAG Manager is a microservice that provides Retrieval-Augmented Generation for VoIPBin documentation.`,
	RunE:  run,
}

func init() {
	// Define flags
	rootCmd.Flags().String("prometheus_endpoint", "/metrics", "URL for the Prometheus metrics endpoint")
	rootCmd.Flags().String("prometheus_listen_address", ":2112", "Address for Prometheus to listen on")
	rootCmd.Flags().String("rabbitmq_address", "amqp://guest:guest@localhost:5672", "Address of the RabbitMQ server")
	rootCmd.Flags().String("openai_api_key", "", "OpenAI API key")
	rootCmd.Flags().String("openai_embedding_model", "text-embedding-3-small", "OpenAI embedding model")
	rootCmd.Flags().String("rag_llm_model", "gpt-4o", "LLM model for answer generation")
	rootCmd.Flags().Int("rag_top_k", 5, "Default number of chunks to retrieve")
	rootCmd.Flags().Int("rag_chunk_max_tokens", 800, "Maximum tokens per chunk")
	rootCmd.Flags().String("gcs_bucket", "", "GCS bucket for embeddings storage")
	rootCmd.Flags().String("gcs_embeddings_path", "rag/embeddings.gob", "GCS path for embeddings file")
	rootCmd.Flags().String("rag_docs_base_path", "", "Base path to document sources")

	// Initialize logging
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)

	// Initialize signal handler
	signal.Notify(chSigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go signalHandler()
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Errorf("Failed to execute command: %v", err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	log := logrus.WithField("func", "run")

	// Initialize configuration
	if err := config.InitConfig(cmd); err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	cfg := *config.Get()

	// Initialize Prometheus
	initProm(cfg.PrometheusEndpoint, cfg.PrometheusListenAddress)

	if err := runService(cfg); err != nil {
		log.Errorf("Run func has finished. err: %v", err)
		return err
	}
	<-chDone
	return nil
}

// signalHandler catches signals and set the done
func signalHandler() {
	sig := <-chSigs
	logrus.Debugf("Received signal. sig: %v", sig)
	chDone <- true
}

// runService initializes and starts the RAG service
func runService(cfg config.Config) error {
	log := logrus.WithField("func", "runService")

	// RabbitMQ connection
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, cfg.RabbitMQAddress)
	sockHandler.Connect()

	// Initialize vector store
	vectorStore := store.NewMemoryStore()

	// Load existing embeddings from disk if available
	if cfg.GCSEmbeddingsPath != "" {
		if err := vectorStore.Load(cfg.GCSEmbeddingsPath); err != nil {
			log.Warnf("Could not load embeddings from disk: %v", err)
		} else {
			stats := vectorStore.Stats()
			log.Infof("Loaded %d chunks from disk", stats.ChunkCount)
		}
	}

	// Initialize OpenAI embedder
	emb := embedder.NewOpenAIEmbedder(cfg.OpenAIAPIKey, cfg.OpenAIEmbeddingModel)

	// Initialize generator
	gen := generator.NewGenerator(cfg.OpenAIAPIKey, cfg.RAGLLMModel)

	// Initialize retriever
	ret := retriever.NewRetriever(emb, vectorStore)

	// Initialize rag handler
	ragH := raghandler.NewRagHandler(ret, gen, emb, vectorStore, cfg.RAGDocsBasePath, cfg.GCSEmbeddingsPath, cfg.RAGTopK)

	// Run listen handler
	if err := runListen(sockHandler, ragH); err != nil {
		return err
	}

	return nil
}

// runListen starts the RPC listen handler
func runListen(sockHandler sockhandler.SockHandler, ragH raghandler.RagHandler) error {
	lh := listenhandler.NewListenHandler(sockHandler, ragH)

	if err := lh.Run(string(commonoutline.QueueNameRagRequest), string(commonoutline.QueueNameDelay)); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}

// initProm initializes prometheus settings
func initProm(endpoint, listen string) {
	http.Handle(endpoint, promhttp.Handler())
	go func() {
		for {
			err := http.ListenAndServe(listen, nil)
			if err != nil {
				logrus.Errorf("Could not start prometheus listener")
				time.Sleep(time.Second * 1)
				continue
			}
			break
		}
	}()
}
