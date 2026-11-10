//go:build freebsd

package zfs

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/zombocoder/go-freebsd-libzfs/errors"
	"github.com/zombocoder/go-freebsd-libzfs/internal/driver"
)

// Limits from the ZFS on-disk ZAP format: a user property name must be
// shorter than zapMaxNameLen bytes and its value shorter than zapMaxValueLen.
const (
	zapMaxNameLen  = 256
	zapMaxValueLen = 8192
)

// UserProperty is the value of a ZFS user property and where it comes from.
// User properties are arbitrary string annotations whose names contain a
// colon, conventionally "<reverse-dns-or-tool>:<key>".
type UserProperty struct {
	Value  string
	Source PropertySource // local, inherited or received
}

func isUserPropertyName(name string) bool {
	return strings.Contains(name, ":")
}

func invalidUserProperty(name, detail string) error {
	return errors.NewZfsError("validate_input", name, errors.ErrCodeInval, 22, detail, nil)
}

// validateUserPropertyName applies the rules of zfs_prop_user(): the name
// contains a colon and consists only of ASCII letters, digits and "-_.:",
// and is shorter than ZAP_MAXNAMELEN bytes.
func validateUserPropertyName(name string) error {
	if !isUserPropertyName(name) {
		return invalidUserProperty(name, "user property name must contain a colon")
	}
	if len(name) >= zapMaxNameLen {
		return errors.NewZfsError("name_validation", name, errors.ErrCodeNameTooLong, 63,
			fmt.Sprintf("user property name must be shorter than %d bytes", zapMaxNameLen), nil)
	}
	for i := 0; i < len(name); i++ {
		ch := name[i]
		switch {
		case ch >= 'a' && ch <= 'z', ch >= '0' && ch <= '9':
		case ch == '-', ch == '_', ch == '.', ch == ':':
		default:
			return invalidUserProperty(name, fmt.Sprintf("invalid character %q in user property name", ch))
		}
	}
	return nil
}

// validateUserProperty checks a user property name and value.
func validateUserProperty(name, value string) error {
	if err := validateUserPropertyName(name); err != nil {
		return err
	}
	if strings.IndexByte(value, 0) >= 0 {
		return invalidUserProperty(name, "user property value must not contain NUL bytes")
	}
	if len(value) >= zapMaxValueLen {
		return invalidUserProperty(name, fmt.Sprintf("user property value must be shorter than %d bytes", zapMaxValueLen))
	}
	return nil
}

// validateUserProperties checks every user property in a property map;
// native properties are left to libzfs.
func validateUserProperties(props map[string]string) error {
	for name, value := range props {
		if isUserPropertyName(name) {
			if err := validateUserProperty(name, value); err != nil {
				return err
			}
		}
	}
	return nil
}

// UserProperties returns all user properties of a dataset (or volume or
// snapshot), including values inherited from ancestors.
func (c *Client) UserProperties(ctx context.Context, datasetName string) (map[string]UserProperty, error) {
	if c.d == nil {
		return nil, fmt.Errorf("client is closed")
	}

	infos, err := c.d.GetUserProperties(ctx, datasetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get user properties: %w", err)
	}

	props := make(map[string]UserProperty, len(infos))
	for name, info := range infos {
		props[name] = UserProperty{Value: info.Value, Source: PropertySource(info.Source.String())}
	}
	return props, nil
}

// GetUserProperty returns the value of one user property. ok is false, with
// a nil error, when the property is not set on the dataset or any ancestor.
//
// Inherited values are returned too, so ok does not mean the dataset itself
// carries the property. Do not use it for ownership checks: use
// GetLocalUserProperty or UserProperties and look at the source.
func (c *Client) GetUserProperty(ctx context.Context, datasetName, name string) (value string, ok bool, err error) {
	if err := validateUserPropertyName(name); err != nil {
		return "", false, err
	}

	props, err := c.UserProperties(ctx, datasetName)
	if err != nil {
		return "", false, err
	}
	p, ok := props[name]
	return p.Value, ok, nil
}

// GetLocalUserProperty is like GetUserProperty but reports ok only when the
// property is set on the dataset itself, not inherited from an ancestor. Use
// it to decide whether a dataset was marked by you.
func (c *Client) GetLocalUserProperty(ctx context.Context, datasetName, name string) (value string, ok bool, err error) {
	if err := validateUserPropertyName(name); err != nil {
		return "", false, err
	}

	props, err := c.UserProperties(ctx, datasetName)
	if err != nil {
		return "", false, err
	}
	p, found := props[name]
	if !found || p.Source != PropertySourceLocal {
		return "", false, nil
	}
	return p.Value, true, nil
}

// FindByUserProperty returns the filesystems and volumes below parent
// (excluding parent itself and snapshots) on which the user property name is
// set locally to value, sorted by name. Inherited values do not match, so
// children of an annotated dataset are not reported just because they inherit
// the value.
//
// The tree is walked once from parent; datasets destroyed during the walk are
// skipped. It fails with errors.ErrDatasetNotFound if parent does not exist.
func (c *Client) FindByUserProperty(ctx context.Context, parent, name, value string) ([]Dataset, error) {
	if c.d == nil {
		return nil, fmt.Errorf("client is closed")
	}
	if err := validateUserPropertyName(name); err != nil {
		return nil, err
	}

	children, err := c.d.ListDescendantUserProps(ctx, parent)
	if err != nil {
		return nil, fmt.Errorf("failed to list datasets below %q: %w", parent, err)
	}

	var found []Dataset
	for _, ch := range children {
		if p, ok := ch.Props[name]; ok && p.Source == driver.PropSourceLocal && p.Value == value {
			found = append(found, Dataset{Name: ch.Name, Type: mapDriverTypeToPublic(ch.Type)})
		}
	}
	sort.Slice(found, func(i, j int) bool { return found[i].Name < found[j].Name })
	return found, nil
}
