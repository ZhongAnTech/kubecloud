package resource

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"kubecloud/common/utils"
	"kubecloud/common/validate"

	"github.com/astaxie/beego"
	"gopkg.in/yaml.v2"
)

const (
	NetworkErrorRatio   = "NetworkErrorRatio"
	LatencyAtQuantileMS = "LatencyAtQuantileMS"
	ResponseCodeRatio   = "ResponseCodeRatio"

	extractorfuncClientIP    = "client.ip"
	extractorfuncRequestHost = "request.host"
	ruleTypePathPrefixStrip  = "PathPrefixStrip"
	ruleTypePathPrefix       = "PathPrefix"
	ruleTypePathStrip        = "PathStrip"
	ruleTypePath             = "Path"

	ingressPrefix = "traefik."
	// ingress annotation
	annotationKubernetesRedirectPermanent   = ingressPrefix + "ingress.kubernetes.io/redirect-permanent"
	annotationKubernetesRedirectRegex       = ingressPrefix + "ingress.kubernetes.io/redirect-regex"
	annotationKubernetesRedirectReplacement = ingressPrefix + "ingress.kubernetes.io/redirect-replacement"
	annotationKubernetesRateLimit           = ingressPrefix + "ingress.kubernetes.io/rate-limit"
	annotationKubernetesBuffering           = ingressPrefix + "ingress.kubernetes.io/buffering"
	annotationKubernetesRuleType            = ingressPrefix + "ingress.kubernetes.io/rule-type"
	annotationKubernetesRewriter            = ingressPrefix + "ingress.kubernetes.io/rewrite-target"
	// service annotation
	annotationKubernetesMaxConnAmount            = ingressPrefix + "ingress.kubernetes.io/max-conn-amount"
	annotationKubernetesMaxConnExtractorFunc     = ingressPrefix + "ingress.kubernetes.io/max-conn-extractor-func"
	annotationKubernetesAffinity                 = ingressPrefix + "ingress.kubernetes.io/affinity"
	annotationKubernetesSessionCookieName        = ingressPrefix + "ingress.kubernetes.io/session-cookie-name"
	annotationKubernetesCircuitBreakerExpression = ingressPrefix + "ingress.kubernetes.io/circuit-breaker-expression"
)

var ruleTypeMap = map[string]interface{}{
	ruleTypePathPrefixStrip: nil,
	ruleTypePathPrefix:      nil,
	ruleTypePathStrip:       nil,
	ruleTypePath:            nil,
}

type ConfigInterface interface {
	get(anno map[string]string)
	set(anno map[string]string, param interface{}) error
	validate() error
}

type affinity struct {
	Affinity          bool   `json:"affinity"`
	SessionCookieName string `json:"session_cookie_name"`
}

type redirect struct {
	Src  string `json:"src"`
	Dest string `json:"dest"`
	host string
}

type breaker struct {
	BreakerType    string  `json:"breaker_type"`
	ThresholdValue float64 `json:"threshold_value"`
}

type connection struct {
	MaxConnAmount int64 `json:"max_conn_amount"`
}

// Rate holds a rate limiting configuration for a specific time period
type rateLimit struct {
	Period  int64 `json:"period"` // unit: ms
	Average int64 `json:"average"`
	Burst   int64 `json:"burst"`
	paths   []string
}

type traefikRate struct {
	Period  time.Duration `yaml:"period"`
	Average int64         `yaml:"average"`
	Burst   int64         `yaml:"burst"`
}

// RateLimit holds a rate limiting configuration for a given frontend
type traefikRateLimit struct {
	RateSet       map[string]*traefikRate `yaml:"rateset"`
	ExtractorFunc string                  `yaml:"extractorfunc"`
}

// Buffering holds request/response buffering configuration/
type buffering struct {
	MaxRequestBodyBytes  int64  `json:"maxRequestBodyBytes,omitempty" yaml:"maxrequestbodybytes"`
	MemRequestBodyBytes  int64  `json:"memRequestBodyBytes,omitempty" yaml:"memrequestbodybytes"`
	MaxResponseBodyBytes int64  `json:"maxResponseBodyBytes,omitempty" yaml:"maxresponsebodybytes"`
	MemResponseBodyBytes int64  `json:"memResponseBodyBytes,omitempty" yaml:"memresponsebodybytes"`
	Retry                int    `json:"retry,omitempty" yaml:"-"`
	RetryExpression      string `json:"-" yaml:"retryexpression"`
}

type ruleType struct {
	RuleType string `json:"rule_type"`
}

type rewriter struct {
	Target string `json:"target"`
}

type IngressConfig struct {
	Protocol   string      `json:"protocol,omitempty"`
	SecretName string      `json:"secret_name,omitempty"`
	Affinity   *affinity   `json:"affinity,omitempty"`
	Breaker    *breaker    `json:"breaker,omitempty"`
	Redirect   *redirect   `json:"redirect,omitempty"`
	Buffering  *buffering  `json:"buffering,omitempty"`
	RateLimit  *rateLimit  `json:"rate_limit,omitempty"`
	Connection *connection `json:"connection,omitempty"`
	RuleType   *ruleType   `json:"rule_type,omitempty"`
	Rewriter   *rewriter   `json:"rewriter,omitempty"`
}

type IngressConfer struct {
	config         IngressConfig
	serviceConfers []ConfigInterface
	ingressConfers []ConfigInterface
}

func NewIngressConfer(config *IngressConfig, paths []string, host string) *IngressConfer {
	confer := &IngressConfer{}
	if config == nil {
		newConfig := IngressConfig{}
		newConfig.Affinity = &affinity{}
		newConfig.Breaker = &breaker{}
		newConfig.Connection = &connection{}
		confer.serviceConfers = []ConfigInterface{newConfig.Connection, newConfig.Affinity, newConfig.Breaker}
		newConfig.Buffering = &buffering{}
		newConfig.Redirect = &redirect{host: host}
		newConfig.RateLimit = &rateLimit{paths: paths}
		newConfig.RuleType = &ruleType{}
		newConfig.Rewriter = &rewriter{}
		confer.ingressConfers = []ConfigInterface{
			newConfig.Buffering,
			newConfig.Redirect,
			newConfig.RateLimit,
			newConfig.RuleType,
			newConfig.Rewriter,
		}
		confer.config = newConfig
		return confer
	}
	confer.config = *config
	if confer.config.Redirect != nil {
		confer.config.Redirect.host = host
	}
	if confer.config.RateLimit != nil {
		confer.config.RateLimit.paths = paths
	}
	getConferList := func(specConfig IngressConfig) []ConfigInterface {
		elem := reflect.ValueOf(&specConfig).Elem()
		confers := []ConfigInterface{}
		for i := 0; i < elem.NumField(); i++ {
			field := elem.Field(i)
			if field.Kind() == reflect.Ptr && !field.IsNil() {
				if confer, ok := field.Interface().(ConfigInterface); ok {
					confers = append(confers, confer)
				}
			}
		}
		return confers
	}
	confer.serviceConfers = getConferList(IngressConfig{
		Affinity:   config.Affinity,
		Breaker:    config.Breaker,
		Connection: config.Connection,
	})
	confer.ingressConfers = getConferList(IngressConfig{
		Redirect:  config.Redirect,
		Buffering: config.Buffering,
		RateLimit: config.RateLimit,
		RuleType:  config.RuleType,
		Rewriter:  config.Rewriter,
	})
	return confer
}

func (confer *IngressConfer) Validate() error {
	confers := []ConfigInterface{}
	confers = append(confers, confer.ingressConfers...)
	confers = append(confers, confer.serviceConfers...)
	errs := []string{}
	for _, confer := range confers {
		if err := confer.validate(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) != 0 {
		return fmt.Errorf(strings.Join(errs, "; "))
	}
	return nil
}

func (confer *IngressConfer) SetIngress(annotation map[string]string) error {
	errs := []string{}
	for _, obj := range confer.ingressConfers {
		if err := obj.set(annotation, nil); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) != 0 {
		return fmt.Errorf(strings.Join(errs, "; "))
	}
	return nil
}

func (confer *IngressConfer) SetService(annotation map[string]string) error {
	errs := []string{}
	for _, obj := range confer.serviceConfers {
		if err := obj.set(annotation, nil); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) != 0 {
		return fmt.Errorf(strings.Join(errs, "; "))
	}
	return nil
}

func (confer *IngressConfer) GetConfig(ingAnno, svcAnno map[string]string) IngressConfig {
	for _, obj := range confer.serviceConfers {
		obj.get(svcAnno)
	}
	for _, obj := range confer.ingressConfers {
		obj.get(ingAnno)
	}
	return confer.config
}

func DeletePathRateLimit(annotation map[string]string, paths []string) {
	rate := getTraefikRateLimit(annotation)
	if rate != nil {
		for _, path := range paths {
			delete(rate.RateSet, utils.GetRootPath(path))
			if len(rate.RateSet) == 0 {
				delete(annotation, annotationKubernetesRateLimit)
				return
			}
		}
		exp, err := yaml.Marshal(rate)
		if err != nil {
			beego.Error(err)
		}
		annotation[annotationKubernetesRateLimit] = string(exp)
	}
}

func getTraefikRateLimit(annotation map[string]string) *traefikRateLimit {
	rateRaw := utils.GetLabelStringValue(annotation, annotationKubernetesRateLimit, "")
	if len(rateRaw) > 0 {
		rlimit := traefikRateLimit{}
		err := yaml.Unmarshal([]byte(rateRaw), &rlimit)
		if err == nil {
			return &rlimit
		} else {
			beego.Error(annotation, err)
		}
	}
	return nil
}

// config get or set
func (config *redirect) get(annotation map[string]string) {
	permanent := utils.GetLabelStringValue(annotation, annotationKubernetesRedirectPermanent, "false")
	if permanent == "true" {
		config.Src = utils.GetLabelStringValue(annotation, annotationKubernetesRedirectRegex, "")
		config.Dest = utils.GetLabelStringValue(annotation, annotationKubernetesRedirectReplacement, "")
	}
}

func (config *redirect) set(annotation map[string]string, param interface{}) error {
	old := &redirect{host: config.host}
	old.get(annotation)
	if config.Src != old.Src || config.Dest != old.Dest {
		if config.Src == "" && config.Dest == "" {
			// close
			delete(annotation, annotationKubernetesRedirectPermanent)
			delete(annotation, annotationKubernetesRedirectRegex)
			delete(annotation, annotationKubernetesRedirectReplacement)
		} else if config.Src != "" && config.Dest != "" {
			annotation[annotationKubernetesRedirectPermanent] = "true"
			annotation[annotationKubernetesRedirectRegex] = config.Src
			annotation[annotationKubernetesRedirectReplacement] = config.Dest
		}
	}
	return nil
}

func (config *redirect) validate() error {
	if !(config.Src == "" && config.Dest == "" || config.Src != "" && config.Dest != "") {
		return fmt.Errorf("redirect configure info is not right!")
	}
	return nil
}

func (config *breaker) get(annotation map[string]string) {
	exp := utils.GetLabelStringValue(annotation, annotationKubernetesCircuitBreakerExpression, "")
	if exp != "" {
		items := strings.Split(exp, ">")
		if len(items) == 2 {
			bType := []byte{}
			for _, c := range []byte(strings.TrimSpace(items[0])) {
				if c == '(' {
					break
				}
				bType = append(bType, c)
			}
			config.BreakerType = string(bType)
			config.ThresholdValue, _ = strconv.ParseFloat(strings.TrimSpace(items[1]), 64)
			if config.BreakerType == NetworkErrorRatio ||
				config.BreakerType == ResponseCodeRatio {
				config.ThresholdValue *= 100
			}
		}
	}
}

func (config *breaker) set(annotation map[string]string, param interface{}) error {
	old := &breaker{}
	old.get(annotation)
	if config.BreakerType != old.BreakerType || config.ThresholdValue != old.ThresholdValue {
		if config.BreakerType == "" || config.ThresholdValue == 0 ||
			(strings.ToLower(config.BreakerType) == strings.ToLower(LatencyAtQuantileMS) &&
				int32(config.ThresholdValue) == 0) {
			// close
			delete(annotation, annotationKubernetesCircuitBreakerExpression)
		} else if config.ThresholdValue > 0 {
			exp := ""
			switch strings.ToLower(config.BreakerType) {
			case strings.ToLower(NetworkErrorRatio):
				exp = fmt.Sprintf("%s() > %v", NetworkErrorRatio, config.ThresholdValue/100)
			case strings.ToLower(ResponseCodeRatio):
				exp = fmt.Sprintf("%s(500, 600, 0, 600) > %v", ResponseCodeRatio, config.ThresholdValue/100)
			case strings.ToLower(LatencyAtQuantileMS):
				exp = fmt.Sprintf("%s(50.0) > %v", LatencyAtQuantileMS, int32(config.ThresholdValue))
			}
			annotation[annotationKubernetesCircuitBreakerExpression] = exp
		}
	}
	return nil
}

func (config *breaker) validate() error {
	if config.ThresholdValue < 0 {
		return fmt.Errorf("threshold value of breaker must be equal or above 0!")
	}
	if strings.ToLower(config.BreakerType) == strings.ToLower(NetworkErrorRatio) {
		if config.ThresholdValue > 100 {
			return fmt.Errorf("network error ratio valid range is [0, 100]!")
		}
	} else if strings.ToLower(config.BreakerType) == strings.ToLower(ResponseCodeRatio) {
		if config.ThresholdValue > 100 {
			return fmt.Errorf("server response code error ratio valid range is [0, 100]!")
		}
	} else if strings.ToLower(config.BreakerType) == strings.ToLower(LatencyAtQuantileMS) {
		return nil
	} else if strings.ToLower(config.BreakerType) != "" {
		return fmt.Errorf("the breaker type(%s) can not be supported!", config.BreakerType)
	}
	return nil
}

func (config *rateLimit) get(annotation map[string]string) {
	rateRaw := utils.GetLabelStringValue(annotation, annotationKubernetesRateLimit, "")
	if len(rateRaw) > 0 {
		rlimit := traefikRateLimit{}
		err := yaml.Unmarshal([]byte(rateRaw), &rlimit)
		if err != nil {
			beego.Error(annotation, err)
		} else {
			if val, ok := rlimit.RateSet[utils.GetRootPath(config.paths[0])]; ok {
				config.Period = int64(val.Period/time.Millisecond) / val.Average
				config.Average = val.Average
				config.Burst = val.Burst
			}
		}
	}
}

func (config *rateLimit) set(annotation map[string]string, param interface{}) error {
	// update
	if config.Average != 0 && config.Period != 0 && config.Burst != 0 {
		// old
		rate := getTraefikRateLimit(annotation)
		if rate == nil {
			rate = &traefikRateLimit{
				RateSet:       make(map[string]*traefikRate),
				ExtractorFunc: extractorfuncClientIP,
			}
		}
		// update
		for _, path := range config.paths {
			rootPath := utils.GetRootPath(path)
			rate.RateSet[rootPath] = &traefikRate{
				Period:  time.Duration(config.Period*config.Average) * time.Millisecond,
				Average: config.Average,
				Burst:   config.Burst,
			}
		}
		exp, err := yaml.Marshal(rate)
		if err != nil {
			return err
		}
		annotation[annotationKubernetesRateLimit] = string(exp)
	} else if config.Average == 0 && config.Period == 0 && config.Burst == 0 {
		// close
		DeletePathRateLimit(annotation, config.paths)
	}
	return nil
}

func (config *rateLimit) validate() error {
	if config.Average < 0 || config.Burst < 0 || config.Period < 0 {
		return fmt.Errorf("ratelimit configure info must be equal or above 0!")
	}
	if !((config.Average != 0 && config.Period != 0 && config.Burst != 0) ||
		(config.Average == 0 && config.Period == 0 && config.Burst == 0)) {
		return fmt.Errorf("ratelimit configure info is not right!")
	}
	return nil
}

func (config *buffering) get(annotation map[string]string) {
	buf := utils.GetLabelStringValue(annotation, annotationKubernetesBuffering, "")
	config.MaxRequestBodyBytes = -1
	config.MemRequestBodyBytes = 10
	config.MaxResponseBodyBytes = -1
	config.MemResponseBodyBytes = 10
	config.Retry = 1
	if len(buf) > 0 {
		err := yaml.Unmarshal([]byte(buf), config)
		if err != nil {
			beego.Error(err)
		} else {
			if config.MaxRequestBodyBytes != -1 {
				config.MaxRequestBodyBytes >>= 20
			}
			if config.MemRequestBodyBytes != -1 {
				config.MemRequestBodyBytes >>= 20
			}
			if config.MaxResponseBodyBytes != -1 {
				config.MaxResponseBodyBytes >>= 20
			}
			if config.MemResponseBodyBytes != -1 {
				config.MemResponseBodyBytes >>= 20
			}
			if config.RetryExpression != "" {
				items := strings.Split(config.RetryExpression, "&&")
				attempts := []byte{}
				for _, item := range items {
					if strings.Contains(item, "Attempts") {
						attempts = []byte(strings.TrimSpace(item))
						break
					}
				}
				if len(attempts) != 0 {
					retry := []byte{}
					for _, item := range attempts {
						if item >= '0' && item <= '9' {
							retry = append(retry, item)
						} else {
							if len(retry) != 0 {
								break
							}
						}
					}
					config.Retry, _ = strconv.Atoi(string(retry))
				}
			}
		}
	}
}

func (config *buffering) set(annotation map[string]string, param interface{}) error {
	oldbuf := &buffering{}
	oldbuf.get(annotation)
	// update
	if config.MemResponseBodyBytes != oldbuf.MemResponseBodyBytes ||
		config.MaxResponseBodyBytes != oldbuf.MaxResponseBodyBytes ||
		config.MemRequestBodyBytes != oldbuf.MemRequestBodyBytes ||
		config.MaxRequestBodyBytes != oldbuf.MaxRequestBodyBytes ||
		config.Retry != oldbuf.Retry {
		if config.Retry <= 0 {
			return fmt.Errorf("retry must be in [1, 5]")
		}
		if config.MemResponseBodyBytes != -1 {
			config.MemResponseBodyBytes *= 1024 * 1024
		}
		if config.MaxResponseBodyBytes != -1 {
			config.MaxResponseBodyBytes *= 1024 * 1024
		}
		if config.MemRequestBodyBytes != -1 {
			config.MemRequestBodyBytes *= 1024 * 1024
		}
		if config.MaxRequestBodyBytes != -1 {
			config.MaxRequestBodyBytes *= 1024 * 1024
		}
		config.RetryExpression = fmt.Sprintf("IsNetworkError() && Attempts() <= %d", config.Retry)
		exp, err := yaml.Marshal(&config)
		if err != nil {
			return err
		}
		annotation[annotationKubernetesBuffering] = string(exp)
	}
	return nil
}

func (config *buffering) validate() error {
	return nil
}

func (config *affinity) get(annotation map[string]string) {
	sticky := utils.GetLabelStringValue(annotation, annotationKubernetesAffinity, "false")
	config.Affinity = false
	if sticky == "true" {
		config.Affinity = true
		config.SessionCookieName = utils.GetLabelStringValue(annotation, annotationKubernetesSessionCookieName, "")
	}
}

func (config *affinity) set(annotation map[string]string, param interface{}) error {
	old := &affinity{}
	old.get(annotation)
	if old.Affinity != config.Affinity {
		if config.Affinity {
			annotation[annotationKubernetesAffinity] = "true"
		} else {
			delete(annotation, annotationKubernetesAffinity)
		}
	}
	if old.SessionCookieName != config.SessionCookieName {
		if config.SessionCookieName != "" {
			annotation[annotationKubernetesSessionCookieName] = config.SessionCookieName
		} else {
			delete(annotation, annotationKubernetesSessionCookieName)
		}
	}
	return nil
}

func (config *affinity) validate() error {
	return nil
}

func (config *connection) get(anno map[string]string) {
	config.MaxConnAmount = utils.GetInt64Value(anno, annotationKubernetesMaxConnAmount, 0)
}

func (config *connection) set(anno map[string]string, param interface{}) error {
	old := &connection{}
	old.get(anno)
	if old.MaxConnAmount != config.MaxConnAmount {
		if config.MaxConnAmount == 0 {
			delete(anno, annotationKubernetesMaxConnAmount)
			delete(anno, annotationKubernetesMaxConnExtractorFunc)
		} else {
			anno[annotationKubernetesMaxConnAmount] = fmt.Sprintf("%v", config.MaxConnAmount)
			anno[annotationKubernetesMaxConnExtractorFunc] = extractorfuncClientIP
		}
	}
	return nil
}

func (config *connection) validate() error {
	if config.MaxConnAmount < 0 {
		return fmt.Errorf("max connection amount must be equle or above 0, 0 means unlimited!")
	}
	return nil
}

func (config *ruleType) get(anno map[string]string) {
	config.RuleType = utils.GetLabelStringValue(anno, annotationKubernetesRuleType, "")
}

func (config *ruleType) set(anno map[string]string, param interface{}) error {
	old := &ruleType{}
	old.get(anno)
	if old.RuleType != config.RuleType {
		if config.RuleType == "" {
			delete(anno, annotationKubernetesRuleType)
		} else {
			anno[annotationKubernetesRuleType] = config.RuleType
		}
	}
	return nil
}

func (config *ruleType) validate() error {
	_, existed := ruleTypeMap[config.RuleType]
	if config.RuleType != "" && !existed {
		return fmt.Errorf("rule type must be %s/%s/%s/%s or emtpy!", ruleTypePathPrefix, ruleTypePathPrefixStrip, ruleTypePath, ruleTypePathStrip)
	}
	return nil
}

func (config *rewriter) get(anno map[string]string) {
	config.Target = utils.GetLabelStringValue(anno, annotationKubernetesRewriter, "")
}

func (config *rewriter) set(anno map[string]string, param interface{}) error {
	old := &rewriter{}
	old.get(anno)
	if old.Target != config.Target {
		if config.Target == "" {
			delete(anno, annotationKubernetesRewriter)
		} else {
			config.Target = utils.AddRootPath(config.Target)
			anno[annotationKubernetesRewriter] = config.Target
		}
	}
	return nil
}

func (config *rewriter) validate() error {
	if len(config.Target) > validate.NormalMaxLen {
		return fmt.Errorf("length of rewrite target must no be larger %v bytes!", validate.NormalMaxLen)
	}
	return nil
}
