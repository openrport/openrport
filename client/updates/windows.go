//+build windows

package updates

import (
	"context"
	"strings"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"github.com/scjalliance/comshim"

	chshare "github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/cloudradar-monitoring/rport/share/models"
)

var packageManagers = []PackageManager{
	NewWindowsPackageManager(),
}

type WindowsPackageManager struct {
}

func NewWindowsPackageManager() *WindowsPackageManager {
	return &WindowsPackageManager{}
}

func (p *WindowsPackageManager) IsAvailable(ctx context.Context) bool {
	return true
}

func (p *WindowsPackageManager) GetUpdatesStatus(ctx context.Context, logger *chshare.Logger) (*models.UpdatesStatus, error) {
	comshim.Add(1)
	defer comshim.Done()

	updates, err := p.listUpdates()
	if err != nil {
		return nil, err
	}
	defer updates.Release()

	summaries := []models.UpdateSummary{}
	securityCount := 0
	err = p.forEach(updates, func(item *ole.IDispatch) error {
		title, err := p.getString(item, "Title")
		if err != nil {
			return err
		}
		description, err := p.getString(item, "Description")
		if err != nil {
			return err
		}
		rebootRequired, err := p.getBool(item, "RebootRequired")
		if err != nil {
			return err
		}

		categoriesVariant, err := oleutil.GetProperty(item, "Categories")
		if err != nil {
			return err
		}
		categories := categoriesVariant.ToIDispatch()
		defer categories.Release()

		isSecurity := false
		err = p.forEach(categories, func(category *ole.IDispatch) error {
			name, err := p.getString(category, "Name")
			if err != nil {
				return err
			}

			if strings.Contains(name, "Security Updates") {
				isSecurity = true
			}

			return nil
		})
		if err != nil {
			return err
		}

		summaries = append(summaries, models.UpdateSummary{
			Title:            title,
			Description:      description,
			IsSecurityUpdate: isSecurity,
			RebootRequired:   rebootRequired,
		})

		if isSecurity {
			securityCount++
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	rebootPending, err := p.checkRebootPending()
	if err != nil {
		return nil, err
	}

	return &models.UpdatesStatus{
		UpdatesAvailable:         len(summaries),
		SecurityUpdatesAvailable: securityCount,
		UpdateSummaries:          summaries,
		RebootPending:            rebootPending,
	}, nil
}

func (p *WindowsPackageManager) checkRebootPending() (bool, error) {
	sysInfo, err := p.newCOMObject("Microsoft.Update.SystemInfo")
	if err != nil {
		return false, err
	}
	defer sysInfo.Release()

	return p.getBool(sysInfo, "RebootRequired")
}

func (p *WindowsPackageManager) listUpdates() (*ole.IDispatch, error) {
	sess, err := p.newCOMObject("Microsoft.Update.Session")
	if err != nil {
		return nil, err
	}
	defer sess.Release()

	searcherVariant, err := oleutil.CallMethod(sess, "CreateUpdateSearcher")
	if err != nil {
		return nil, err
	}
	searcher := searcherVariant.ToIDispatch()
	defer searcher.Release()

	resultsVariant, err := oleutil.CallMethod(searcher, "Search", "IsHidden=0 and IsInstalled=0")
	if err != nil {
		return nil, err
	}
	results := resultsVariant.ToIDispatch()
	defer results.Release()

	updatesVariant, err := oleutil.GetProperty(results, "Updates")
	if err != nil {
		return nil, err
	}

	return updatesVariant.ToIDispatch(), nil
}

func (p *WindowsPackageManager) newCOMObject(name string) (*ole.IDispatch, error) {
	unknown, err := oleutil.CreateObject(name)
	if err != nil {
		return nil, err
	}
	defer unknown.Release()

	disp, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return nil, err
	}
	return disp, nil
}

func (p *WindowsPackageManager) forEach(list *ole.IDispatch, f func(item *ole.IDispatch) error) error {
	countVariant, err := oleutil.GetProperty(list, "Count")
	if err != nil {
		return err
	}
	defer countVariant.Clear()
	count := int(countVariant.Val)

	for i := 0; i < count; i++ {
		itemVariant, err := oleutil.GetProperty(list, "item", i)
		if err != nil {
			return err
		}
		item := itemVariant.ToIDispatch()
		defer item.Release()

		if err := f(item); err != nil {
			return err
		}
	}
	return nil
}

func (p *WindowsPackageManager) getString(obj *ole.IDispatch, key string) (string, error) {
	variant, err := oleutil.GetProperty(obj, key)
	if err != nil {
		return "", err
	}
	defer variant.Clear()

	return variant.ToString(), nil
}

func (p *WindowsPackageManager) getBool(obj *ole.IDispatch, key string) (bool, error) {
	variant, err := oleutil.GetProperty(obj, key)
	if err != nil {
		return false, err
	}
	defer variant.Clear()

	return variant.Value().(bool), nil
}
