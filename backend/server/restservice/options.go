package restservice

import (
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/models"
)

func flattenDHCPOptions(optionSpace string, restOptions []*models.DHCPOption) (options []dbmodel.DHCPOption) {
	for _, restOption := range restOptions {
		option := dbmodel.DHCPOption{
			AlwaysSend:  restOption.AlwaysSend,
			Code:        uint16(restOption.Code),
			Encapsulate: restOption.Encapsulate,
		}
		if len(optionSpace) > 0 {
			option.Space = optionSpace
		}
		for _, restField := range restOption.Fields {
			field := dbmodel.DHCPOptionField{
				FieldType: restField.FieldType,
			}
			for _, value := range restField.Values {
				field.Values = append(field.Values, value)
			}
			option.Fields = append(option.Fields, field)
		}
		if len(restOption.Options) > 0 {
			suboptions := flattenDHCPOptions(option.Encapsulate, restOption.Options)
			options = append(options, suboptions...)
		}
		options = append(options, option)
	}
	return
}
