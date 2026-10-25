package lab

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"hw2-mongo/internal/service/catalog"
	"hw2-mongo/internal/service/dataset"
	"hw2-mongo/internal/service/failover"
	ordersService "hw2-mongo/internal/service/orders"
	"hw2-mongo/internal/service/purchase"
	"hw2-mongo/internal/service/report"
	"hw2-mongo/internal/storage/mongo/fixtures"
	ordersStorage "hw2-mongo/internal/storage/mongo/orders"
	productsStorage "hw2-mongo/internal/storage/mongo/products"
)

const (
	StatusNotRun   = "not_run"
	StatusPending  = "pending"
	StatusRunning  = "running"
	StatusPassed   = "passed"
	StatusFailed   = "failed"
	StatusOutdated = "outdated"
)

type Env struct {
	Client   *mongo.Client
	Catalog  *catalog.Service
	Orders   *ordersService.Service
	Report   *report.Service
	Purchase *purchase.Service
	Failover *failover.Service
	Dataset  *dataset.Service
	Products *productsStorage.Storage
	OrdersSt *ordersStorage.Storage

	ServerVersion string

	ResetMode string
}

type ScenarioFunc func(ctx context.Context, r *Runner) error

type ScenarioError struct {
	Message string `bson:"message"`
	Code    any    `bson:"code"`
}

type ScenarioEntry struct {
	Status         string         `bson:"status"`
	PreviousStatus string         `bson:"previousStatus,omitempty"`
	SolutionHash   string         `bson:"solutionHash,omitempty"`
	StartedAt      *time.Time     `bson:"startedAt,omitempty"`
	FinishedAt     *time.Time     `bson:"finishedAt,omitempty"`
	Server         string         `bson:"server,omitempty"`
	Error          *ScenarioError `bson:"error,omitempty"`
	Checks         []string       `bson:"checks"`
	Artifacts      map[string]any `bson:"artifacts"`
}

type Results struct {
	Dataset   string                    `bson:"dataset"`
	UpdatedAt time.Time                 `bson:"updatedAt"`
	Scenarios map[string]*ScenarioEntry `bson:"scenarios"`
}

type Runner struct {
	env     *Env
	path    string
	results *Results
	current *ScenarioEntry
}

func NewRunner(env *Env, resultsPath string) *Runner {
	return &Runner{env: env, path: resultsPath, results: loadResults(resultsPath)}
}

var ScenarioNames = []string{"data", "index", "purchase", "failover"}

func (r *Runner) Run(ctx context.Context, names []string) error {
	r.markOutdated()
	if len(names) == 1 && names[0] == "reset" {
		r.clearScenarioStatuses()
	}

	for _, name := range names {
		scenario, ok := scenarios[name]
		if !ok {
			return fmt.Errorf("неизвестная команда: %s", name)
		}

		entry := &ScenarioEntry{
			Status: StatusRunning, StartedAt: nowPtr(), SolutionHash: solutionHash(),
			Checks: []string{}, Artifacts: map[string]any{},
		}
		r.results.Scenarios[name] = entry
		r.current = entry

		if err := r.persist(); err != nil {
			return err
		}

		Logf("RUN %s {dataset: %s}", name, fixtures.DatasetVersion)

		runErr := r.runScenario(ctx, name, entry, scenario)

		entry.FinishedAt = nowPtr()
		if runErr == nil {
			entry.Status = StatusPassed
		} else {
			entry.Status = StatusFailed
			entry.Error = &ScenarioError{Message: runErr.Error()}
		}

		if err := r.persist(); err != nil {
			return err
		}

		if runErr != nil {
			return fmt.Errorf("сценарий %s: %w", name, runErr)
		}

		Logf("COMPLETE %s {passed: %d, results: %s}", name, len(entry.Checks), r.path)
	}

	return nil
}

func (r *Runner) runScenario(
	ctx context.Context,
	name string,
	entry *ScenarioEntry,
	scenario ScenarioFunc,
) error {
	var build struct {
		Version string `bson:"version"`
	}
	if err := r.env.Client.Database("admin").
		RunCommand(ctx, bson.D{{Key: "buildInfo", Value: 1}}).Decode(&build); err != nil {
		return fmt.Errorf("buildInfo: %w", err)
	}

	entry.Server = build.Version
	if build.Version != r.env.ServerVersion {
		return fmt.Errorf("нужна MongoDB %s, получена %s", r.env.ServerVersion, build.Version)
	}

	return scenario(ctx, r)
}

func (r *Runner) markOutdated() {
	currentHash := solutionHash()
	for _, entry := range r.results.Scenarios {
		if entry.SolutionHash != currentHash &&
			(entry.Status == StatusPassed || entry.Status == StatusFailed) {
			entry.PreviousStatus = entry.Status
			entry.Status = StatusOutdated
		}
	}
}

func (r *Runner) clearScenarioStatuses() {
	for _, name := range ScenarioNames {
		r.results.Scenarios[name] = &ScenarioEntry{Status: StatusNotRun, Checks: []string{}}
	}
}

func (r *Runner) check(name string, condition bool) error {
	if !condition {
		return fmt.Errorf("FAIL: %s", name)
	}

	r.current.Checks = append(r.current.Checks, name)
	if err := r.persist(); err != nil {
		return err
	}

	fmt.Println("PASS " + name)

	return nil
}

func (r *Runner) equal(name string, actual, expected any) error {
	actualJSON, err := normalized(actual)
	if err != nil {
		return fmt.Errorf("normalization actual (%s): %w", name, err)
	}

	expectedJSON, err := normalized(expected)
	if err != nil {
		return fmt.Errorf("normalization expected (%s): %w", name, err)
	}

	return r.check(name, actualJSON == expectedJSON)
}

func (r *Runner) save(name string, value any) error {
	r.current.Artifacts[name] = value

	return r.persist()
}

func (r *Runner) persist() error {
	r.results.UpdatedAt = time.Now()

	data, err := bson.MarshalExtJSONIndent(r.results, true, false, "", "  ")
	if err != nil {
		return fmt.Errorf("MarshalExtJSON: %w", err)
	}

	dir := filepath.Dir(r.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("MkdirAll: %w", err)
	}

	tmp := r.path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("WriteFile: %w", err)
	}

	if err := os.Rename(tmp, r.path); err != nil {
		return fmt.Errorf("Rename: %w", err)
	}

	return nil
}

func loadResults(path string) *Results {
	results := &Results{
		Dataset:   fixtures.DatasetVersion,
		Scenarios: map[string]*ScenarioEntry{},
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return results
	}

	var loaded Results
	if err := bson.UnmarshalExtJSON(data, true, &loaded); err != nil {
		return results
	}

	if loaded.Scenarios == nil {
		loaded.Scenarios = map[string]*ScenarioEntry{}
	}

	results.Dataset = fixtures.DatasetVersion
	results.Scenarios = loaded.Scenarios
	results.UpdatedAt = loaded.UpdatedAt

	return results
}

func solutionHash() string {
	exe, err := os.Executable()
	if err != nil {
		return "unknown"
	}

	file, err := os.Open(exe)
	if err != nil {
		return "unknown"
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "unknown"
	}

	return fmt.Sprintf("%x", hash.Sum(nil))
}

func nowPtr() *time.Time {
	now := time.Now().UTC()

	return &now
}
