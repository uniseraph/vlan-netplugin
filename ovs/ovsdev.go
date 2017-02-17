package ovs

import "github.com/weaveworks/go-odp/odp"

func  CreateDatapath(name string) (odp.DatapathHandle,error) {

	dpif , err :=  odp.NewDpif()
	if err!=nil{
		return odp.DatapathHandle{},err
	}
	defer dpif.Close()


	handler , err := dpif.CreateDatapath(name)
	if err != nil {
		if odp.IsDatapathNameAlreadyExistsError(err) {
			return dpif.LookupDatapath(name)
		} else {
			return odp.DatapathHandle{} , err
		}
	}

	return handler , nil

}


//
//func DeleteDatapath(name string) error {
//
//	dpif, err := odp.NewDpif()
//	if err != nil {
//		return err
//	}
//	defer dpif.Close()
//
//	dp, err := dpif.LookupDatapath(name)
//	if err!=nil{
//		return err
//	}
//
//
//	err := dpif.D
//	if err != nil {
//		return printErr("%s", err)
//	}
//
//	return true
//}