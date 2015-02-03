package tagger

import (
	"fmt"
	"strings"
)

// TODO: Investigate negate/invert filter and it's sql equivalent

// Filter provides an interface to filter files based on their tags
type Filter interface {
	fmt.Stringer
	// TODO: This interface might not  if databases engines want to optimize filtering
	Matches(t []Tag) bool
}

// NameFilter filters tags on their names
type NameFilter struct {
	Name string
}

// Matches check if the filter matches the given tags
func (n NameFilter) Matches(tags []Tag) bool {
	for _, tag := range tags {
		if tag.Name() == n.Name {
			return true
		}
	}
	return false
}

// Comparator describes a way to compare two integer values
type Comparator int

// Definitions of various comparinson operators
const (
	Invalid Comparator = iota
	Equals
	NotEquals
	LessThan
	GreaterThan
	LessThanOrEqual
	GreaterThanOrEqual
)

func ComparatorFromString(val string) Comparator {
	switch val {
	case "==":
		return Equals
	case "!=":
		return NotEquals
	case "<":
		return LessThan
	case ">":
		return GreaterThan
	case "<=":
		return LessThanOrEqual
	case ">=":
		return GreaterThanOrEqual
	default:
		return Invalid
	}
}

// ComparinsonFilter filters value tags based on their value
type ComparinsonFilter struct {
	Name     string
	Value    int
	Function Comparator
}

// Matches check if the filter matches the given tags
func (c ComparinsonFilter) Matches(tags []Tag) bool {
	for _, tag := range tags {
		if tag.Name() == c.Name {
			if !tag.HasValue() {
				return false
			}

			switch c.Function {
			case Equals:
				return tag.Value() == c.Value

			case NotEquals:
				return tag.Value() != c.Value

			case LessThan:
				return tag.Value() < c.Value

			case GreaterThan:
				return tag.Value() > c.Value

			case LessThanOrEqual:
				return tag.Value() <= c.Value

			case GreaterThanOrEqual:
				return tag.Value() >= c.Value
			}
		}
	}

	return false
}

// AndFilter allows the joining of two or more filters, all which must match
type AndFilter struct {
	Filters []Filter
}

// Matches check if the filter matches the given tags
func (a AndFilter) Matches(tags []Tag) bool {
	for _, filter := range a.Filters {
		if !filter.Matches(tags) {
			return false
		}
	}

	return true
}

// OrFilter allows the joining of two or more filters, one of which must match
type OrFilter struct {
	Filters []Filter
}

// Matches check if the filter matches the given tags
func (o OrFilter) Matches(tags []Tag) bool {
	for _, filter := range o.Filters {
		if filter.Matches(tags) {
			return true
		}
	}

	return false
}

// Debuggg

func (c Comparator) String() string {
	switch c {
	case Equals:
		return "=="

	case NotEquals:
		return "!="

	case LessThan:
		return "<"

	case GreaterThan:
		return ">"

	case LessThanOrEqual:
		return "<="

	case GreaterThanOrEqual:
		return ">="
	}
	return "INVALID"
}

func (n NameFilter) String() string {
	return fmt.Sprintf("%s", n.Name)
}

func (c ComparinsonFilter) String() string {
	return fmt.Sprintf("%s %s %d", c.Name, c.Function, c.Value)
}

func (a AndFilter) String() string {
	subs := make([]string, 0)
	for _, f := range a.Filters {
		subs = append(subs, f.String())
	}
	return fmt.Sprintf("(%s)", strings.Join(subs, ", "))
}

func (a OrFilter) String() string {
	subs := make([]string, 0)
	for _, f := range a.Filters {
		subs = append(subs, f.String())
	}
	return fmt.Sprintf("(%s)", strings.Join(subs, ", "))
}
