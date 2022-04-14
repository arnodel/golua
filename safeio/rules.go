package safeio

import (
	"strings"
)

// FSAccessRule knows how to allow or deny actions on a certain file path.
type FSAccessRule interface {
	GetFSAccessEffect(path string, requested FSAction) (allowed FSAction, denied FSAction)
}

// FSAccessRuleset allows grouping rules together.
type FSAccessRuleset struct {
	Rules []FSAccessRule
}

// GetFSAccessEffect returns Deny if any of its rules returns Deny, otherwise
// returns Allow if any of its rules returns Allow, otherwise returns None.
func (s FSAccessRuleset) GetFSAccessEffect(path string, requested FSAction) (allowed FSAction, denied FSAction) {
	for _, r := range s.Rules {
		a, d := r.GetFSAccessEffect(path, requested)
		allowed |= a
		denied |= d
	}
	return
}

// FSAccessRulechain chains rules, i.e. for an action to be allowed it needs to
// be allowed by all members of the chain.
type FSAccessRulechain struct {
	Rules []FSAccessRule
}

func (r FSAccessRulechain) GetFSAccessEffect(path string, requested FSAction) (allowed FSAction, denied FSAction) {
	allowed, denied = AllFileActions, AllFileActions
	for _, r := range r.Rules {
		a, d := r.GetFSAccessEffect(path, requested)
		allowed &= a
		denied |= d
	}
	return
}

type PrefixFSAccessRule struct {
	Prefix         string
	AllowedActions FSAction
	DeniedActions  FSAction
}

func (r PrefixFSAccessRule) GetFSAccessEffect(path string, actions FSAction) (allowed FSAction, denied FSAction) {
	if !strings.HasPrefix(path, r.Prefix) {
		// If the path does start with r.Prefix, there is no effect from this rule.
		return 0, 0
	}
	return actions & r.AllowedActions, actions & r.DeniedActions
}

// MergeFSAccessRules returns an FSAccessRule representing all the rules passed
// in.  It discards nil rules and flattens rulesets.
func MergeFSAccessRules(rules ...FSAccessRule) FSAccessRule {
	var mergedRules []FSAccessRule
	for _, r := range rules {
		if r == nil {
			continue
		}
		if s, ok := r.(FSAccessRuleset); ok {
			mergedRules = append(mergedRules, s.Rules...)
		} else {
			mergedRules = append(mergedRules, r)
		}
	}
	switch len(mergedRules) {
	case 0:
		return nil
	case 1:
		return mergedRules[0]
	default:
		return FSAccessRuleset{Rules: mergedRules}
	}
}

// MergeFSAccessRules returns an FSAccessRule chaining all the rules passed in.
// It discards nil rules.
func ChainFSAccessRules(rules ...FSAccessRule) FSAccessRule {
	var chain []FSAccessRule
	for _, r := range rules {
		if r == nil {
			continue
		}
		chain = append(chain, r)
	}
	switch len(chain) {
	case 0:
		return nil
	case 1:
		return chain[0]
	default:
		return FSAccessRulechain{Rules: chain}
	}
}
