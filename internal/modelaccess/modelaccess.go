// Package modelaccess is the platform's model-access policy: which of the
// models the Gateway actually serves are open to platform users, what
// name and description users see, and whether a user's request may use a
// model. It is one rule read by the admin's model page, the user's model
// plaza, user API keys and the Gateway itself, so a model shown as open
// is one the Gateway lets a user call, and a closed one is refused even
// by a key that listed it before.
//
// A model with no rule is closed. The rule is kept in model-access.json
// beside magpie's settings; the models themselves come from the existing
// providers and catalog, never from a second directory of their own.
package modelaccess

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/yetone/magpie/internal/edit"
	"github.com/yetone/magpie/internal/filelock"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/settings"
)

// Rule is one model's access as the admin set it.
type Rule struct {
	Open        bool      `json:"open,omitempty"`
	Name        string    `json:"name,omitempty"`
	Description string    `json:"description,omitempty"`
	UpdatedAt   time.Time `json:"updatedAt,omitzero"`
}

// Model is an actual Gateway model with the policy applied: the admin's
// page sees every one, the user's plaza only the open ones.
type Model struct {
	ID          string    `json:"id"` // the id a request names, provider/model or group/<id>
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Provider    string    `json:"provider"`
	ProviderID  string    `json:"providerId"`
	Protocols   []string  `json:"protocols"`
	Available   bool      `json:"available"`
	Context     int       `json:"context,omitempty"`
	Images      bool      `json:"images,omitempty"`
	Kind        string    `json:"kind,omitempty"` // "group" for a routing group
	// PublicID is the id users see and call: a routing group's own id
	// without "group/" ("deepseek-v4.1-flash"), which the Gateway takes to
	// that group; empty for anything else, whose ID is already public.
	PublicID  string    `json:"publicId,omitempty"`
	Open      bool      `json:"open"`
	UpdatedAt time.Time `json:"updatedAt,omitzero"`
}

// PublicIDOf is the id a user is shown and calls for m: m.PublicID when it
// has one, else m.ID.
func PublicIDOf(m Model) string {
	if m.PublicID != "" {
		return m.PublicID
	}
	return m.ID
}

// Path is the policy file, beside magpie's own settings.
func Path() string { return filepath.Join(settings.Dir(), "model-access.json") }

type policyFile struct {
	// GroupsOnly keeps provider models out of the user's plaza and out of
	// a user key's reach; only open routing groups are offered there.
	GroupsOnly bool            `json:"groupsOnly,omitempty"`
	Models     map[string]Rule `json:"models"`
}

var mu = filelock.Mutex{Path: func() string { return Path() + ".lock" }}

// cache keeps the parsed policy between requests; every Gateway request
// checks one. A changed file (its size or time) is read again at once.
var cache struct {
	sync.Mutex
	path       string
	mod        time.Time
	size       int64
	groupsOnly bool
	rules      map[string]Rule
}

func loadPolicy() (policyFile, error) {
	path := Path()
	info, statErr := os.Stat(path)
	cache.Lock()
	if cache.path == path && statErr == nil && cache.mod.Equal(info.ModTime()) && cache.size == info.Size() {
		out := policyFile{GroupsOnly: cache.groupsOnly, Models: cache.rules}
		cache.Unlock()
		return out, nil
	}
	cache.Unlock()
	if errors.Is(statErr, os.ErrNotExist) {
		return policyFile{Models: map[string]Rule{}}, nil
	}
	if statErr != nil {
		return policyFile{}, statErr
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return policyFile{}, err
	}
	var f policyFile
	if err := json.Unmarshal(b, &f); err != nil {
		return policyFile{}, err
	}
	if f.Models == nil {
		f.Models = map[string]Rule{}
	}
	cache.Lock()
	cache.path, cache.mod, cache.size, cache.rules, cache.groupsOnly = path, info.ModTime(), info.Size(), f.Models, f.GroupsOnly
	cache.Unlock()
	return f, nil
}

func loadRules() (map[string]Rule, error) {
	p, err := loadPolicy()
	return p.Models, err
}

func savePolicy(p policyFile) error {
	if err := os.MkdirAll(filepath.Dir(Path()), 0o700); err != nil {
		return err
	}
	if p.Models == nil {
		p.Models = map[string]Rule{}
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.OpenFile(Path(), os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Chmod(Path(), 0o600); err != nil {
		return err
	}
	if err := edit.WriteAtomic(Path(), append(b, '\n')); err != nil {
		return err
	}
	cache.Lock()
	cache.path = ""
	cache.Unlock()
	return nil
}

func saveRules(rules map[string]Rule) error {
	p, err := loadPolicy()
	if err != nil {
		return err
	}
	p.Models = rules
	return savePolicy(p)
}

// GroupsOnly is whether the user's plaza and user keys are held to open
// routing groups alone. Provider models stay for agents and administrators.
func GroupsOnly() bool {
	p, err := loadPolicy()
	if err != nil {
		return false
	}
	return p.GroupsOnly
}

// SetGroupsOnly switches what platform users are offered: every open model
// (false) or open routing groups alone (true). Provider models keep their
// own open rules.
func SetGroupsOnly(on bool) error {
	mu.Lock()
	defer mu.Unlock()
	p, err := loadPolicy()
	if err != nil {
		return err
	}
	p.GroupsOnly = on
	return savePolicy(p)
}

// Allows is whether id's rule says open. A missing rule is closed. A rule
// left for a model the Gateway no longer serves still says open here, but
// that id is never in AllModels, OpenIDs or OpenModels, so no user key or
// plaza is granted it.
func Allows(id string) bool {
	rules, err := loadRules()
	if err != nil {
		return false // a policy that can't be read opens nothing
	}
	return rules[id].Open
}

// AllowsAny is Allows for any of ids.
func AllowsAny(ids ...string) bool {
	rules, err := loadRules()
	if err != nil {
		return false
	}
	for _, id := range ids {
		if id != "" && rules[id].Open {
			return true
		}
	}
	return false
}

// AllowsModel is whether p's model is open, by each id it goes by: the
// vendor's id, the user's listed id and the id a request names. With
// GroupsOnly, platform users are held to routing groups alone.
func AllowsModel(p provider.Provider, model string) bool {
	if GroupsOnly() {
		return false
	}
	ids := []string{p.ID + "/" + model}
	if own := p.OwnModelID(model); own != model {
		ids = append(ids, p.ID+"/"+own)
	}
	if listed := p.ListedModelID(model); listed != model {
		ids = append(ids, p.ID+"/"+listed)
	}
	return AllowsAny(ids...)
}

// OpenIDs are the ids of the open models the Gateway actually serves now,
// as a user key's explicit model list is made from them. With GroupsOnly,
// only open routing groups are offered.
func OpenIDs() []string {
	models := AllModels()
	groupsOnly := GroupsOnly()
	out := make([]string, 0, len(models))
	for _, m := range models {
		if m.Open && (!groupsOnly || m.Kind == "group") {
			out = append(out, m.ID)
		}
	}
	return out
}

// AllModels is every model the Gateway serves now, with the policy's open
// flag and the admin's name and description. A model the policy names but
// the Gateway no longer serves is not here. A routing group is listed by
// the id a request names, "group/<id>", with its members' protocols and
// availability; the admin opens it as one model.
func AllModels() []Model {
	shown, _ := provider.CatalogFor("")
	rules, _ := loadRules()
	seen := map[string]bool{}
	out := make([]Model, 0, len(shown))
	for _, e := range shown {
		if e.Provider == nil || e.Provider.DecideOnly() || e.Provider.DecidesModel(e.Model) {
			continue
		}
		if seen[e.ID] {
			continue
		}
		seen[e.ID] = true
		if e.Group != "" {
			if m, ok := groupModelOf(e, rules); ok {
				out = append(out, m)
			}
			continue
		}
		p := *e.Provider
		out = append(out, modelOf(e, p, rules))
	}
	slices.SortFunc(out, func(a, b Model) int { return strings.Compare(a.ID, b.ID) })
	return out
}

// OpenModels is AllModels' open ones, for the user's plaza: every open
// one, or with GroupsOnly the open routing groups alone.
func OpenModels() []Model {
	out := make([]Model, 0)
	groupsOnly := GroupsOnly()
	for _, m := range AllModels() {
		if m.Open && (!groupsOnly || m.Kind == "group") {
			out = append(out, m)
		}
	}
	return out
}

// groupModelOf is a routing group as the directory shows it: the group
// name, its members' protocols, its least context and whether any member
// can answer now. Its price is not one number, so none is invented.
func groupModelOf(e provider.Entry, rules map[string]Rule) (Model, bool) {
	g, ms, ok := provider.FindGroup(e.ID)
	if !ok || len(ms) == 0 {
		return Model{}, false
	}
	r := rules[e.ID]
	name := strings.TrimSpace(r.Name)
	if name == "" {
		name = cmpOr(e.Name, g.Name, g.ID)
	}
	protos := make([]string, 0, 4)
	available := false
	for _, m := range ms {
		if m.Provider.On() {
			available = true
		}
		apis := m.Provider.APIs(m.Model)
		if len(apis) == 0 {
			apis = m.Provider.Speaks()
		}
		for _, api := range apis {
			protos = append(protos, string(api))
		}
	}
	if len(protos) == 0 && e.Provider != nil {
		for _, api := range e.Provider.Speaks() {
			protos = append(protos, string(api))
		}
	}
	protos = unique(protos)
	public := strings.TrimPrefix(e.ID, provider.GroupPrefix)
	return Model{ID: e.ID, Name: name, Description: strings.TrimSpace(r.Description), Protocols: protos,
		Available: available, Context: e.Context, Images: e.Images, Kind: "group", PublicID: public,
		Open: r.Open, UpdatedAt: r.UpdatedAt}, true
}

func modelOf(e provider.Entry, p provider.Provider, rules map[string]Rule) Model {
	r := rules[e.ID]
	name := strings.TrimSpace(r.Name)
	if name == "" {
		name = cmpOr(e.Name, e.Model)
	}
	protos := make([]string, 0, 4)
	for _, api := range p.APIs(e.Model) {
		protos = append(protos, string(api))
	}
	if len(protos) == 0 {
		for _, api := range p.ListedAPIs(e.Model) {
			protos = append(protos, string(api))
		}
	}
	if len(protos) == 0 {
		// a model typed in before a list gave its APIs is still called on
		// the protocols its provider is set up for
		for _, api := range p.Speaks() {
			protos = append(protos, string(api))
		}
	}
	protos = unique(protos)
	return Model{ID: e.ID, Name: name, Description: strings.TrimSpace(r.Description), Provider: p.Name, ProviderID: p.ID,
		Protocols: protos, Available: p.On(), Context: e.Context, Images: e.Images, Open: r.Open, UpdatedAt: r.UpdatedAt}
}

func cmpOr(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func unique(in []string) []string {
	out := in[:0]
	for _, s := range in {
		if s != "" && !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	return out
}

// Update changes id's access and, when given, the name and description
// users see. A nil field keeps its rule as it is, so opening a model doesn't
// rewrite what the admin wrote. id must be a model or routing group the
// Gateway actually serves now ("provider/model" or "group/<id>").
func Update(id string, open *bool, name, description *string) (Model, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Model{}, errors.New("name the model")
	}
	var cleanName, cleanDesc string
	if name != nil {
		cleanName = strings.TrimSpace(*name)
		if utf8.RuneCountInString(cleanName) > 80 {
			return Model{}, errors.New("a display name is 80 characters at most")
		}
	}
	if description != nil {
		cleanDesc = strings.TrimSpace(*description)
		if utf8.RuneCountInString(cleanDesc) > 500 {
			return Model{}, errors.New("a description is 500 characters at most")
		}
	}
	found := false
	for _, m := range AllModels() {
		if m.ID == id {
			found = true
			break
		}
	}
	if !found {
		return Model{}, errors.New("the Gateway doesn't serve that model now")
	}
	mu.Lock()
	defer mu.Unlock()
	rules, err := loadRules()
	if err != nil {
		return Model{}, err
	}
	r := rules[id]
	if open != nil {
		r.Open = *open
	}
	if name != nil {
		r.Name = cleanName
	}
	if description != nil {
		r.Description = cleanDesc
	}
	r.UpdatedAt = time.Now().UTC()
	rules[id] = r
	if err := saveRules(rules); err != nil {
		return Model{}, err
	}
	// read it back through the same directory the pages use, so the answer
	// is the effective name even when the admin cleared their own
	for _, m := range AllModels() {
		if m.ID == id {
			return m, nil
		}
	}
	return Model{}, errors.New("the Gateway doesn't serve that model now")
}

// Set opens or closes id, and sets the name and description users see.
func Set(id string, open bool, name, description string) (Model, error) {
	return Update(id, &open, &name, &description)
}
