package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"digital.vasic.challenges/pkg/challenge"
	"digital.vasic.challenges/pkg/registry"
	"digital.vasic.challenges/pkg/report"
	"digital.vasic.challenges/pkg/runner"
)

func main() {
	var (
		skipBoot   = flag.Bool("skip-boot", false, "Skip service health checks and assume services are running")
		timeout    = flag.Duration("timeout", 10*time.Minute, "Total challenge suite timeout")
		_          = flag.Int("parallel", 4, "Max parallel challenges") // reserved for future use
		resultsDir = flag.String("results-dir", "results", "Directory for challenge results")
		logsDir    = flag.String("logs-dir", "logs", "Directory for challenge logs")
		verbose    = flag.Bool("verbose", false, "Enable verbose logging")
	)
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nShutdown signal received, cancelling...")
		cancel()
	}()

	cfg := LoadConfig()
	cfg.ResultsDir = *resultsDir
	cfg.LogsDir = *logsDir

	fmt.Println("=== HelixFlow Comprehensive Challenge Suite ===")
	fmt.Println(cfg.String())

	logger := &cliLogger{verbose: *verbose}

	if !*skipBoot {
		fmt.Println("\n--- Pre-flight Health Checks ---")
		if err := preflightHealthChecks(ctx, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Pre-flight checks failed: %v\n", err)
			fmt.Println("Hint: Start services with ./start_all_services.sh or use -skip-boot")
			os.Exit(1)
		}
		fmt.Println("All services are healthy.")
	}

	reg := registry.NewRegistry()
	registerAllChallenges(reg, cfg)

	fmt.Printf("\nRegistered %d challenges\n", reg.Count())

	run := runner.NewRunner(
		runner.WithRegistry(reg),
		runner.WithLogger(logger),
		runner.WithTimeout(2*time.Minute),
		runner.WithResultsDir(*resultsDir),
	)
	// Override to sequential execution to prevent token state interference between challenges
	_ = run

	fmt.Println("\n--- Executing Challenges ---")
	results, err := run.RunAll(ctx, &challenge.Config{
		ResultsDir: *resultsDir,
		LogsDir:    *logsDir,
		Timeout:    2 * time.Minute,
		Verbose:    *verbose,
		Environment: map[string]string{
			"API_GATEWAY_URL":    cfg.APIGatewayURL,
			"AUTH_SERVICE_URL":   cfg.AuthServiceURL,
			"INFERENCE_POOL_URL": cfg.InferencePoolURL,
			"MONITORING_URL":     cfg.GetMonitoringURL(),
			"TEST_USERNAME":      cfg.Username,
			"TEST_PASSWORD":      cfg.Password,
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Runner error: %v\n", err)
	}

	passed, failed, skipped := 0, 0, 0
	for _, r := range results {
		switch r.Status {
		case challenge.StatusPassed:
			passed++
		case challenge.StatusFailed, challenge.StatusError, challenge.StatusTimedOut, challenge.StatusStuck:
			failed++
		case challenge.StatusSkipped:
			skipped++
		}
	}

	fmt.Printf("\n=== Results ===\n")
	fmt.Printf("Passed:  %d\n", passed)
	fmt.Printf("Failed:  %d\n", failed)
	fmt.Printf("Skipped: %d\n", skipped)
	fmt.Printf("Total:   %d\n", len(results))

	mdReporter := report.NewMarkdownReporter(*resultsDir)
	mdReport, _ := mdReporter.GenerateMasterSummary(results)
	reportPath := *resultsDir + "/challenge-report.md"
	if err := os.WriteFile(reportPath, mdReport, 0644); err == nil {
		fmt.Printf("\nReport written to: %s\n", reportPath)
	}

	// Save individual JSON reports
	jsonReporter := report.NewJSONReporter(*resultsDir, true)
	for _, r := range results {
		jsonData, _ := jsonReporter.GenerateReport(r)
		_ = os.WriteFile(fmt.Sprintf("%s/%s.json", *resultsDir, r.ChallengeID), jsonData, 0644)
	}
	fmt.Printf("Individual JSON reports written to: %s/\n", *resultsDir)

	if failed > 0 {
		fmt.Println("\n❌ CHALLENGE SUITE FAILED")
		os.Exit(1)
	}
	fmt.Println("\n✅ ALL CHALLENGES PASSED")
}

// cliLogger implements challenge.Logger with simple console output.
type cliLogger struct {
	verbose bool
}

func (l *cliLogger) Info(msg string, args ...any)  { fmt.Printf("[INFO]  %s\n", fmt.Sprintf(msg, args...)) }
func (l *cliLogger) Warn(msg string, args ...any)  { fmt.Printf("[WARN]  %s\n", fmt.Sprintf(msg, args...)) }
func (l *cliLogger) Error(msg string, args ...any) { fmt.Fprintf(os.Stderr, "[ERROR] %s\n", fmt.Sprintf(msg, args...)) }
func (l *cliLogger) Debug(msg string, args ...any) {
	if l.verbose {
		fmt.Printf("[DEBUG] %s\n", fmt.Sprintf(msg, args...))
	}
}
func (l *cliLogger) Close() error { return nil }

func registerAllChallenges(reg *registry.DefaultRegistry, cfg *HelixFlowConfig) {
	// Health & Discovery
	reg.Register(NewHealthCheckChallenge(cfg))

	// Auth Lifecycle
	reg.Register(NewAuthRegisterChallenge(cfg))
	reg.Register(NewAuthLoginChallenge(cfg))
	reg.Register(NewAuthTokenRefreshChallenge(cfg))
	reg.Register(NewAuthTokenRevocationChallenge(cfg))
	reg.Register(NewAuthJWTValidationChallenge(cfg))

	// API Gateway
	reg.Register(NewGatewayModelsChallenge(cfg))
	reg.Register(NewGatewayChatCompletionsChallenge(cfg))
	reg.Register(NewGatewayRateLimitingChallenge(cfg))

	// WebSocket
	reg.Register(NewWebSocketStreamingChallenge(cfg))

	// Inference Pool
	reg.Register(NewInferenceModelExecutionChallenge(cfg))
	reg.Register(NewInferenceStreamingChallenge(cfg))

	// Monitoring
	reg.Register(NewMonitoringMetricsChallenge(cfg))
	reg.Register(NewMonitoringAlertsChallenge(cfg))

	// Security
	reg.Register(NewSecurityTLSChallenge(cfg))
	reg.Register(NewSecurityRateLimitingChallenge(cfg))
	reg.Register(NewSecurityBruteForceChallenge(cfg))

	// End-to-End
	reg.Register(NewE2EFullChatFlowChallenge(cfg))
	reg.Register(NewE2ETokenLifecycleChallenge(cfg))
}
