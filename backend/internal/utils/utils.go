package utils

import (
	"encoding/json"
	"fmt"
	"reflect"

	"URLS/internal/utils/strconvext"
)

// PrettyJSONPrint 輸出格式化的 JSON 結構 (Only for debug)
func PrettyJSONPrint(v any) {
	bs, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		rv := reflect.ValueOf(v)
		fmt.Printf("json.Marshal %s failed, err=%s\n", rv.Type(), err)
	}
	fmt.Println(strconvext.B2S(bs))
}
