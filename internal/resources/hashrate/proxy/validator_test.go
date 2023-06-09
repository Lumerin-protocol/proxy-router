package proxy

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/proxy/stratumv1_message"
)

func TestValidator(t *testing.T) {
	/* from the configure response */
	version_mask := "00c00000"

	/* from the subscribe response */
	en2_size := uint(8)
	en1 := "00"

	/* from the last set_difficulty */
	job_diff := uint64(65536)

	/* a notify message */
	notify_raw := `{"id":1,"method":"mining.notify","params":["B8gWrWpoC","e1d647c06683e348640c2ee97046f92c37ffd061000f29c80000000000000000","01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff6403a955092cfabe6d6d839353feb00d1d58a0b27e7805f375ef27ab3c307d84454ea2896a502535049110000000f09f909f00114d696e656420627920746974616e646576000000000000000000000000000000000000000000000000000005","040c892e4c000000001976a914c825a1ecf2a6830c4401620c3a16f1995057c2ab88ac00000000000000002f6a24aa21a9ed4459c4318ef71610fb022c0f64165f3b7b4ff14066f962a9763300962031472b08000000000000000000000000000000002c6a4c2952534b424c4f434b3aa37b0123223e60cd77512f5414f4fa5ded2b9a96238710f802b88829001ec31b0000000000000000266a24b9e11b6dd1d08c2f8671e5a5b7135e700c85fda519e1b429029f9971ea708dc2ab3326f4a95d9f39",["9b4966ab64ba1fbe14ab0a1ff61fdcf6776f9389fca6aef1b9212dc5f365c3e0","b4e080d85d02528db5c2c7b3f781231db2fd427b1489398f4362c6da33ab0f38","c7e2a9ba1e6ad650d6772f6000e41a3541d9ba1bdfad4e82b043ff48369c647c","fc3dc9e23acaae1974f9acf58bf944be7dc969b1594dc965837605166820f90d","2d313cf9d28b5fe3d25dae100a716aca911fb092a5dde5b80b478cc6874ac5de","62eb6e02b0db580df83d8dd81586468fa51fc2786ff19a22007052841025abcf","9de122f40275fdc071d098fdede2f378548b17e717256a100afa94e00a220940","daaa45136bbfea5c9fe7ac56d5aaf1125f086903d8ddaa240cd01e469ad6fb33","431e9c3fcd3ab1feadf478bd0891ce6321e7246547918fc6a8e3e3b41bad4c32","e99473b768537099a7f86b69e9aedc8e1419fd3aae6a8cea3db4a3ca233a1873","cdc8915b844d93b2bc71ed27f6f9c27b33750d678f556c4b8f70f1f259521a9f","0d855f6e6f2f688e4d6a0620b3394ef4d2cdbc3a7ca297810d53652ac708174c"],"20000000","171465f2","5e14b65f",true]}`
	job, _ := stratumv1_message.ParseMiningNotify([]byte(notify_raw))

	/* a submit response */
	submit_raw := `{"id":2, "method": "mining.submit", "params": ["titandev.961a83b2087b491", "B8gWrWpoC", "0919000000000000", "5e14b65f", "dce1b752", "00000000"]}`
	submit, _ := stratumv1_message.ParseMiningSubmit([]byte(submit_raw))

	/* GO */
	diff, ok := Validate(en1, en2_size, job_diff, version_mask, job, submit)
	var meets string
	if ok {
		meets = "meets"
	} else {
		meets = "does NOT meet"
	}
	fmt.Printf("Result diff (%d) %s difficulty target (%d)\n", diff, meets, job_diff)
	assert.Truef(t, ok, "Result diff doesn't meet difficulty target")
}
