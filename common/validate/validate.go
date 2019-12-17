package validate

import (
	"encoding/json"
	"kubecloud/common"
	"kubecloud/common/keyword"
	"regexp"
	"strings"
	"unicode"

	kubevalidation "k8s.io/apimachinery/pkg/util/validation"
)

const (
	restrictedNameChars  = `[a-z0-9]+(?:[-][a-z0-9]+)*`
	versionChars         = `[a-zA-Z0-9]+(?:[-._][a-zA-Z0-9]+)*`
	restricteddomainName = `([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,6}`
	restrictedIP         = `((25[0-5]|2[0-4]\d|((1\d{2})|([1-9]?\d)))\.){3}(25[0-5]|2[0-4]\d|((1\d{2})|([1-9]?\d)))`
)

const (
	VersionMinLen      = 0
	VersionMaxLen      = 63
	NameMinLen         = 2
	NameMaxLen         = 63
	SubPathMinLen      = 0
	SubPathMaxLen      = 64
	DomainNameMinLen   = 1
	DomainNameMaxLen   = 255
	MAX_DEPLOY_TIMEOUT = 120
	DescriptionMaxLen  = 255
	NormalMaxLen       = 255
	NodePortMin        = 30000
	NodePortMax        = 32767
)

var LabelsStrMaxLenMap = map[string]int{
	keyword.K8S_RESOURCE_TYPE_NODE: 2048,
	keyword.K8S_RESOURCE_TYPE_APP:  64,
}

// 验证中文字符
func IsChineseChar(s string) bool {
	for _, v := range s {
		if unicode.Is(unicode.Han, v) {
			return true
		}
	}
	return false
}

// 验证字符长度
func IsIllegalLength(s string, min int, max int) bool {
	if min == -1 {
		return (len(s) > max)
	}
	if max == -1 {
		return (len(s) <= min)
	}
	return (len(s) < min || len(s) > max)
}

// 正则表达式验证字符合法性
func Restricted(s, regdata string) bool {
	validName := regexp.MustCompile(`^` + regdata + `$`)
	legal := validName.MatchString(s)
	return legal
}

// 系统保留的关键字
func IsReservedBuName(s string) error {
	stringList := []string{
		"all", "default", "kube-system", "kube-public", "admin",
	}
	for _, str := range stringList {
		if s == str {
			return common.NewBadRequest().
				SetCode("ReservedKeyword").
				SetMessage(`can not use system reserved keyword "%v"`, s)
		}
	}
	return nil
}

// 字符串验证
func ValidateString(s string) error {
	if IsIllegalLength(s, NameMinLen, NameMaxLen) {
		return common.NewBadRequest().
			SetCode("InvalidStringLength").
			SetMessage("invalid string length, acceptable range is [%v,%v]", NameMinLen, NameMaxLen)
	}
	if !Restricted(s, restrictedNameChars) {
		return common.NewBadRequest().
			SetCode("InvalidString").
			SetMessage(`invalid string "%s", must start with lower alpha, number beginning and ending, and '-' can be used for separating`, s)
	}
	return nil
}

func ValidateDomainName(domain string) error {
	if IsIllegalLength(domain, DomainNameMinLen, DomainNameMaxLen) {
		return common.NewBadRequest().
			SetCode("InvalidDomainLength").
			SetMessage("invalid domain length, acceptable range is [%v,%v]", DomainNameMinLen, DomainNameMaxLen)
	}
	if !Restricted(domain, restricteddomainName) {
		return common.NewBadRequest().
			SetCode("InvalidDomainName").
			SetMessage(`invalid domain name "%s", it must match "%s"`, domain, restricteddomainName)
	}
	return nil
}

func ValidateSubPath(subpath string) error {
	if IsIllegalLength(subpath, SubPathMinLen, SubPathMaxLen) {
		return common.NewBadRequest().
			SetCode("InvalidSubPath").
			SetMessage("invalid subpath length, acceptable range is [%v,%v]", SubPathMinLen, SubPathMaxLen)
	}
	return nil
}

func ValidatePortNum(port int32) error {
	if port < 1 || port > 65535 {
		return common.NewBadRequest().
			SetCode("InvalidPort").
			SetMessage(`invalid port "%v", acceptable range is [1, 65535]`, port)
	}
	return nil
}

func ValidateNodePortNum(nodePort int32) error {
	if nodePort != 0 && (nodePort < NodePortMin || nodePort > NodePortMax) {
		return common.NewBadRequest().
			SetCode("InvalidNodePort").
			SetMessage(`invalid node port "%v", acceptable range is [%v, %v], or 0`, nodePort, NodePortMin, NodePortMax)
	}
	return nil
}

func ValidateIP(ip string) error {
	if !Restricted(ip, restrictedIP) {
		return common.NewBadRequest().
			SetCode("InvalidIP").
			SetMessage(`invalid IP address "%v", it must match "%s"`, ip, restrictedIP)
	}
	return nil
}

func ValidateDescription(description string) error {
	if IsIllegalLength(description, 0, DescriptionMaxLen) {
		return common.NewBadRequest().
			SetCode("InvalidDescription").
			SetMessage("invalid description length, acceptable range is [0,%v]", DescriptionMaxLen)
	}
	return nil
}

func ValidateAppVersion(ver string) error {
	if IsIllegalLength(ver, VersionMinLen, VersionMaxLen) {
		return common.NewBadRequest().
			SetCode("InvalidAppVersionLength").
			SetMessage("invalid app version length, acceptable range is [%v,%v]", VersionMinLen, VersionMaxLen)
	}
	if ver != "" && !Restricted(ver, versionChars) {
		return common.NewBadRequest().
			SetCode("InvalidAppVersion").
			SetMessage(`invalid app version "%v", must be named with alpha or number beginning and ending, '.','-' can be used for separating`, ver)
	}
	return nil
}

func ValidateLabels(object string, labels map[string]string) error {
	labelStr, err := json.Marshal(labels)
	if err != nil {
		return err
	}
	if len(labelStr) > LabelsStrMaxLenMap[object] {
		return common.NewBadRequest().
			SetCode("InvalidLabelsLength").
			SetMessage("invalid labels length, acceptable range is [%v,%v]", 0, LabelsStrMaxLenMap[object])
	}
	for key, value := range labels {
		if errs := kubevalidation.IsQualifiedName(key); len(errs) != 0 {
			return common.NewBadRequest().
				SetCode("InvalidLabelKey").
				SetMessage(`invalid label key "%v:%s"`, key, strings.Join(errs, "; "))
		}
		if key == keyword.LABEL_APPNAME_KEY ||
			key == keyword.LABEL_PODVERSION_KEY ||
			key == keyword.LABEL_APPVERSION_KEY {
			return common.NewBadRequest().
				SetCode("InvalidLabelKey").
				SetMessage("label key cant be system keyword 'app' or 'version' or 'appversion'")
		}
		if errs := kubevalidation.IsValidLabelValue(value); len(errs) != 0 {
			return common.NewBadRequest().
				SetCode("InvalidLabelValue").
				SetMessage(`invalid label value "%v:%s"`, value, strings.Join(errs, "; "))
		}
	}
	return nil
}
