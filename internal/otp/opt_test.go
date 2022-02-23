package otp

import (
	"fmt"
	"github.com/xlzd/gotp"
	"testing"
	"time"
)

func TestOtp(t *testing.T) {
	totp := gotp.NewDefaultTOTP("4S62BZNFXXSZLCRO")

	totpURL := totp.ProvisioningUri("guanyiyao", "secure-proxy")
	fmt.Println(totpURL)

	if !totp.Verify("", int(time.Now().Unix())) {
		t.Errorf("not match")
	}
}
