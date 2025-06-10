package charger

import (
	"fmt"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

const (
	aristonBaseURL = "https://www.ariston-net.remotethermo.com/api/v2"
)

// Ariston charger implementation
type Ariston struct {
	*request.Helper
	uri       string
	user      string
	pass      string
	deviceID  string
	userAgent string
	token     string
	cache     time.Duration
	statusG   util.Cacheable[bool]
}

func init() {
	registry.Add("ariston", NewAristonFromConfig)
}

// NewAristonFromConfig creates an Ariston charger from generic config
func NewAristonFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI       string
		User      string
		Password  string
		DeviceID  string
		UserAgent string
		Cache     time.Duration
	}{
		UserAgent: "evcc/1.0",
		Cache:     time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, fmt.Errorf("missing uri")
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	if cc.DeviceID == "" {
		return nil, fmt.Errorf("missing device_id")
	}

	return NewAriston(cc.URI, cc.User, cc.Password, cc.DeviceID, cc.UserAgent, cc.Cache)
}

// NewAriston creates an Ariston charger
func NewAriston(uri, user, password, deviceID, userAgent string, cache time.Duration) (*Ariston, error) {
	log := util.NewLogger("ariston").Redact(user, password)

	c := &Ariston{
		Helper:    request.NewHelper(log),
		uri:       util.DefaultScheme(uri, "http"),
		user:      user,
		pass:      password,
		deviceID:  deviceID,
		userAgent: userAgent,
		cache:     cache,
	}

	// setup cached values
	c.statusG = util.ResettableCached(c.getStatus, c.cache)

	// authenticate
	if err := c.authenticate(); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	return c, nil
}

// authenticate performs login and stores the token
func (c *Ariston) authenticate() error {
	data := map[string]string{
		"usr": c.user,
		"pwd": c.pass,
	}

	uri := fmt.Sprintf("%s/accounts/login", aristonBaseURL)
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), map[string]string{
		"User-Agent": c.userAgent,
	})
	if err != nil {
		return err
	}

	var res struct {
		Token string `json:"token"`
	}
	if err = c.DoJSON(req, &res); err != nil {
		return err
	}

	c.token = res.Token
	return nil
}

// reset cache
func (c *Ariston) reset() {
	c.statusG.Reset()
}

// getStatus retrieves the current status
func (c *Ariston) getStatus() (bool, error) {
	uri := fmt.Sprintf("%s/remote/plants/%s/features", aristonBaseURL, c.deviceID)
	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"User-Agent":    c.userAgent,
		"ar.authToken": c.token,
	})
	if err != nil {
		return false, err
	}

	var res struct {
		Boost bool `json:"boost"`
	}
	if err = c.DoJSON(req, &res); err != nil {
		return false, err
	}

	return res.Boost, nil
}

// Status implements the api.Charger interface
func (c *Ariston) Status() (api.ChargeStatus, error) {
	boost, err := c.statusG.Get()
	if err != nil {
		return api.StatusNone, err
	}

	if boost {
		return api.StatusC, nil
	}
	return api.StatusA, nil
}

// Enabled implements the api.Charger interface
func (c *Ariston) Enabled() (bool, error) {
	return c.statusG.Get()
}

// Enable implements the api.Charger interface
func (c *Ariston) Enable(enable bool) error {
	defer c.reset()

	uri := fmt.Sprintf("%s/velis/slpPlantData/%s/boost", aristonBaseURL, c.deviceID)
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(enable), map[string]string{
		"User-Agent":    c.userAgent,
		"ar.authToken": c.token,
	})
	if err != nil {
		return err
	}

	var res struct {
		Success bool `json:"success"`
	}
	if err = c.DoJSON(req, &res); err != nil {
		return err
	}

	if !res.Success {
		return fmt.Errorf("failed to set boost mode")
	}

	return nil
}

// MaxCurrent implements the api.Charger interface
func (c *Ariston) MaxCurrent(current int64) error {
	// Ariston water heaters don't support current control
	return api.ErrNotAvailable
} 