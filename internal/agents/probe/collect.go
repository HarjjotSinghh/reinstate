package probe

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/version"
)

const (
	maxWalkEntries  = 4096
	maxFilesOpened  = 64
	maxLineBytes    = 8 << 10
	maxTreeRows     = 64
	maxVersionBytes = 4 << 10
	defaultTimeout  = 10 * time.Second
)

// Options bounds one probe run and injects test fakes.
//
// Timeout is the budget for a single agent, not for the whole run. A probe on
// a machine with a dozen agents installed spawns a dozen `--version`
// subprocesses, and one slow harness must not be able to discard everything
// the probe already learned about the others.
type Options struct {
	LookPath   func(string) (string, error)
	RunVersion func(ctx context.Context, name string, args []string) (string, error)
	Now        func() time.Time
	Version    string
	Timeout    time.Duration
	MaxFiles   int
	MaxOpens   int
}

// Collect builds one AGENT-PROBE-V1 artifact for descriptors.
func Collect(ctx context.Context, env agents.Env, descriptors []agents.Descriptor, opts Options) (Artifact, error) {
	if opts.Timeout <= 0 {
		opts.Timeout = defaultTimeout
	}
	if opts.MaxFiles <= 0 {
		opts.MaxFiles = maxWalkEntries
	}
	if opts.MaxOpens <= 0 {
		opts.MaxOpens = maxFilesOpened
	}
	now := time.Now().UTC()
	if opts.Now != nil {
		now = opts.Now().UTC()
	}
	ver := version.Version
	if opts.Version != "" {
		ver = opts.Version
	}
	home, err := env.HomeDir()
	if err != nil {
		return Artifact{}, err
	}
	out := Artifact{
		Schema:           Schema,
		GeneratedAt:      now.Format(time.RFC3339),
		Platform:         currentPlatform(),
		ReinstateVersion: ver,
		Agents:           make([]Agent, 0, len(descriptors)),
	}
	for _, d := range descriptors {
		// Only the caller giving up aborts the run. A per-agent deadline is
		// recorded on that agent's record and the walk continues.
		if err := ctx.Err(); err != nil {
			return Artifact{}, err
		}
		out.Agents = append(out.Agents, probeOneAgent(ctx, env, home, d, opts))
	}
	return out, nil
}

func probeOneAgent(ctx context.Context, env agents.Env, home agents.HomeDir, d agents.Descriptor, opts Options) Agent {
	agentCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()
	rec := probeAgent(agentCtx, env, home, d, opts)
	if agentCtx.Err() != nil && ctx.Err() == nil {
		rec.TimedOut = true
	}
	return rec
}

func currentPlatform() Platform {
	return Platform{
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		DeviceClass: agents.CurrentOS(),
	}
}

func probeAgent(ctx context.Context, env agents.Env, home agents.HomeDir, d agents.Descriptor, opts Options) Agent {
	rec := Agent{
		Key:           d.Key,
		DisplayName:   d.DisplayName,
		DeclaredTier:  d.Tier.String(),
		RootEnv:       d.Storage.RootEnv,
		Tree:          []TreeNode{},
		NameShapes:    []NameShape{},
		FirstLineKeys: map[string][]string{},
	}
	if rec.RootEnv != "" {
		rec.RootEnvSet = strings.TrimSpace(env.Lookup(rec.RootEnv)) != ""
	}

	candidates, resolvedAbs, resolved := resolveCandidates(env, home, d)
	if candidates == nil {
		candidates = []CandidateRoot{}
	}
	rec.CandidateRoots = candidates
	rec.ResolvedRoot = resolved

	name := executableName(d)
	if name != "" {
		look := opts.LookPath
		if look == nil {
			look = exec.LookPath
		}
		if path, err := look(name); err == nil && strings.TrimSpace(path) != "" {
			rec.ExecutableOnPath = true
			rec.VersionRaw = readVersion(ctx, name, d, opts)
		}
	}

	if resolvedAbs != "" {
		tree, shapes, keys := walkRoot(ctx, resolvedAbs, accountName(home), d.Storage.Excluded, opts)
		// A resolved but empty root walks to nothing. Assigning the nil results
		// straight through dropped the initialized empty slices and produced an
		// artifact that failed AGENT-PROBE-V1 validation.
		if tree != nil {
			rec.Tree = tree
		}
		if shapes != nil {
			rec.NameShapes = shapes
		}
		if keys != nil {
			rec.FirstLineKeys = keys
		}
	}
	return rec
}

func executableName(d agents.Descriptor) string {
	if d.Native != nil && d.Native.Executable != "" {
		return d.Native.Executable
	}
	if len(d.Process.Images) > 0 {
		return d.Process.Images[0]
	}
	return ""
}

func readVersion(ctx context.Context, name string, d agents.Descriptor, opts Options) string {
	args := []string{"--version"}
	if d.Version != nil && len(d.Version.Args) > 0 {
		args = d.Version.Args
	}
	runner := opts.RunVersion
	if runner == nil {
		runner = runVersionCommand
	}
	raw, err := runner(ctx, name, args)
	if err != nil {
		return ""
	}
	raw = strings.TrimSpace(raw)
	if raw == "" || looksLikePath(raw) {
		return ""
	}
	return raw
}

func runVersionCommand(ctx context.Context, name string, args []string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var buf bytes.Buffer
	cmd.Stdout = &limitedWriter{remaining: maxVersionBytes, buf: &buf}
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

type limitedWriter struct {
	remaining int
	buf       *bytes.Buffer
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	if w.remaining <= 0 {
		return len(p), nil
	}
	if len(p) > w.remaining {
		_, _ = w.buf.Write(p[:w.remaining])
		w.remaining = 0
		return len(p), nil
	}
	n, err := w.buf.Write(p)
	w.remaining -= n
	return n, err
}

func resolveCandidates(env agents.Env, home agents.HomeDir, d agents.Descriptor) ([]CandidateRoot, string, *RelativeRoot) {
	var declared []agents.Root
	if d.Storage.Roots != nil {
		declared = d.Storage.Roots(home)
	}
	candidates := []CandidateRoot{}
	var resolvedAbs string
	var resolved *RelativeRoot
	osName := agents.CurrentOS()
	for _, root := range declared {
		if !root.Matches(osName) {
			continue
		}
		rel := relativize(home.String(), root.Path)
		exists, marker := inspectRoot(root.Path, d.Storage.Marker)
		candidates = append(candidates, CandidateRoot{
			RelativeTo:    rel.RelativeTo,
			Suffix:        rel.Suffix,
			Exists:        exists,
			MarkerPresent: marker,
		})
		// Discovery is marker-gated. An explicit RootEnv or fixture root below
		// is an instruction and is trusted; a home-directory guess is not.
		if resolved == nil && exists && marker {
			copyRel := rel
			resolved = &copyRel
			resolvedAbs = root.Path
		}
	}

	if envPath := strings.TrimSpace(env.Lookup(d.Storage.RootEnv)); d.Storage.RootEnv != "" && envPath != "" {
		exists, marker := inspectRoot(envPath, d.Storage.Marker)
		rel := relativize(home.String(), envPath)
		if rel.RelativeTo != "home" {
			rel = RelativeRoot{RelativeTo: "env", Suffix: ""}
		}
		// An explicit RootEnv outranks a home-directory guess, the same way
		// FixtureRoot does below. Requiring resolved == nil here meant that a
		// tester who pointed the variable at a sanitized root still had their
		// real home tree walked and reported.
		if exists && (d.Storage.Marker == "" || marker) {
			resolved = &rel
			resolvedAbs = envPath
		}
	}
	if env.FixtureRoot != "" {
		exists, marker := inspectRoot(env.FixtureRoot, d.Storage.Marker)
		rel := relativize(home.String(), env.FixtureRoot)
		if rel.RelativeTo != "home" {
			rel = RelativeRoot{RelativeTo: "fixture", Suffix: ""}
		}
		if exists && (d.Storage.Marker == "" || marker) {
			resolved = &rel
			resolvedAbs = env.FixtureRoot
		}
	}
	return candidates, resolvedAbs, resolved
}

func inspectRoot(path, marker string) (exists, markerPresent bool) {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return false, false
	}
	if strings.TrimSpace(marker) == "" {
		return true, false
	}
	_, err = os.Stat(filepath.Join(path, filepath.FromSlash(marker)))
	return true, err == nil
}

func relativize(home, path string) RelativeRoot {
	home = filepath.Clean(home)
	path = filepath.Clean(path)
	if rel, err := filepath.Rel(home, path); err == nil && !strings.HasPrefix(rel, "..") {
		return RelativeRoot{RelativeTo: "home", Suffix: filepath.ToSlash(rel)}
	}
	return RelativeRoot{RelativeTo: "other", Suffix: normalizeComponent(filepath.Base(path))}
}

func looksLikePath(value string) bool {
	if strings.Contains(value, "/Users/") || strings.Contains(value, "/home/") ||
		strings.Contains(value, `:\Users\`) {
		return true
	}
	fields := strings.Fields(value)
	return len(fields) > 0 && filepath.IsAbs(fields[0])
}

// accountName is the operating-system account name, taken from the home
// directory's last component on every supported platform.
func accountName(home agents.HomeDir) string {
	base := strings.TrimSpace(filepath.Base(filepath.Clean(home.String())))
	if base == "." || base == string(filepath.Separator) {
		return ""
	}
	return base
}

func walkRoot(ctx context.Context, root, user string, excluded []string, opts Options) ([]TreeNode, []NameShape, map[string][]string) {
	type dirAgg struct {
		children []int
		samples  int
	}
	type fileAgg struct {
		sizes []int64
	}
	type shapeAgg struct {
		shape string
		n     int
	}
	dirs := map[string]*dirAgg{}
	files := map[string]*fileAgg{}
	shapes := map[string]*shapeAgg{}
	keys := map[string][]string{}
	opened := 0
	walked := 0

	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		walked++
		if walked > opts.MaxFiles {
			return errors.New("walk bound")
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		if rel == "." {
			return nil
		}
		if isExcluded(rel, entry.Name(), excluded) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		components := splitSlash(filepath.ToSlash(rel))
		normalized := make([]string, len(components))
		for i, c := range components {
			normalized[i] = redactUser(normalizeComponent(c), user)
		}
		tp := treePath(normalized)
		if entry.IsDir() {
			agg := dirs[tp]
			if agg == nil {
				agg = &dirAgg{}
				dirs[tp] = agg
			}
			agg.samples++
			if kids, err := os.ReadDir(path); err == nil {
				agg.children = append(agg.children, len(kids))
			}
		} else {
			agg := files[tp]
			if agg == nil {
				agg = &fileAgg{}
				files[tp] = agg
			}
			if info, err := entry.Info(); err == nil {
				agg.sizes = append(agg.sizes, info.Size())
			}
			if opened < opts.MaxOpens && shouldSampleKeys(entry.Name()) {
				if sample, ok := readFirstLineKeys(path); ok {
					if _, exists := keys[tp]; !exists {
						keys[tp] = sample
					}
					opened++
				}
			}
		}
		if len(normalized) > 0 {
			parent := strings.Join(append(append([]string{}, treePath(normalized[:len(normalized)-1])), "*"), "/")
			if len(normalized) == 1 {
				parent = "*"
			}
			shape := shapeOf(normalized[len(normalized)-1])
			if strings.HasPrefix(shape, "<") || strings.Contains(shape, "<") {
				cur := shapes[parent]
				if cur == nil {
					shapes[parent] = &shapeAgg{shape: shape, n: 1}
				} else {
					cur.n++
				}
			}
		}
		return nil
	})

	var tree []TreeNode
	for path, agg := range dirs {
		node := TreeNode{Path: path, Kind: "dir", SampleCount: agg.samples}
		if len(agg.children) > 0 {
			node.Children = medianInt(agg.children)
		}
		if strings.Contains(path, "*") {
			node.SampleCount = agg.samples
		} else {
			node.SampleCount = 0
			if len(agg.children) == 1 {
				node.Children = agg.children[0]
			}
		}
		tree = append(tree, node)
	}
	for path, agg := range files {
		node := TreeNode{Path: path, Kind: "file", Count: len(agg.sizes)}
		if len(agg.sizes) > 0 {
			node.MedianBytes = medianInt64(agg.sizes)
		}
		tree = append(tree, node)
	}
	// Path alone is not a total order: a dir node and a file node can normalize
	// to the same path, and sort.Slice is not stable, so their relative order
	// varied per run. That made the probe non-reproducible and, because the
	// slice is truncated to maxTreeRows below, changed which rows survived.
	sort.Slice(tree, func(i, j int) bool { return treeNodeLess(tree[i], tree[j]) })
	if len(tree) > maxTreeRows {
		tree = tree[:maxTreeRows]
	}

	var named []NameShape
	for path, agg := range shapes {
		named = append(named, NameShape{Path: path, Shape: agg.shape, Samples: agg.n})
	}
	sort.Slice(named, func(i, j int) bool { return nameShapeLess(named[i], named[j]) })

	if keys == nil {
		keys = map[string][]string{}
	}
	return tree, named, keys
}

func shouldSampleKeys(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".json", ".jsonl":
		return true
	default:
		return false
	}
}

func readFirstLineKeys(path string) ([]string, bool) {
	file, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer func() { _ = file.Close() }()
	limited := io.LimitReader(file, int64(maxLineBytes)+1)
	reader := bufio.NewReader(limited)
	line, err := reader.ReadBytes('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, false
	}
	line = bytes.TrimSpace(line)
	if len(line) == 0 || len(line) > maxLineBytes || !json.Valid(line) {
		return nil, false
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(line, &obj) != nil {
		return nil, false
	}
	names := make([]string, 0, len(obj))
	for key := range obj {
		names = append(names, key)
	}
	sort.Strings(names)
	return names, true
}

func medianInt(values []int) int {
	if len(values) == 0 {
		return 0
	}
	cp := append([]int(nil), values...)
	sort.Ints(cp)
	return cp[len(cp)/2]
}

func medianInt64(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}
	cp := append([]int64(nil), values...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	return cp[len(cp)/2]
}

// treeNodeLess is a total order over TreeNode. Every field participates so
// that two nodes compare equal only when they are identical, keeping
// AGENT-PROBE-V1 output byte-reproducible across runs.
func treeNodeLess(a, b TreeNode) bool {
	if a.Path != b.Path {
		return a.Path < b.Path
	}
	if a.Kind != b.Kind {
		return a.Kind < b.Kind
	}
	if a.Children != b.Children {
		return a.Children < b.Children
	}
	if a.Count != b.Count {
		return a.Count < b.Count
	}
	if a.MedianBytes != b.MedianBytes {
		return a.MedianBytes < b.MedianBytes
	}
	return a.SampleCount < b.SampleCount
}

// nameShapeLess is a total order over NameShape, for the same reason.
func nameShapeLess(a, b NameShape) bool {
	if a.Path != b.Path {
		return a.Path < b.Path
	}
	if a.Shape != b.Shape {
		return a.Shape < b.Shape
	}
	return a.Samples < b.Samples
}
