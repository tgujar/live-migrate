package main

import (
	"fmt"
	"sort"
)

// func scheduleMigrationOld(vm_container_map map[string]map[string]float64, target float64) map[string]string {
	
// 	vm_total := make(map[string]float64)
// 	vm_mapping := make(map[string]string)
// 	total_vms := len(vm_container_map)
// 	vms := make([]string, 0, len(vm_container_map))

// 	vm_index := make(map[string]int)

// 	vms_overused_queue := make([]string, 0)

// 	count := 0
// 	for key, element := range vm_container_map{
// 		vm_total[key] = 0.0
// 		for _, cpu_perc := range element{
// 			vm_total[key] = vm_total[key] + cpu_perc
// 		}
// 		if (vm_total[key] > target){
// 			vms_overused_queue = append(vms_overused_queue, key)
// 		}
// 		vms = append(vms, key)
// 		vm_index[key] = count
// 		count++
// 	}

// 	fmt.Println("Printing VM's totals initial:")
// 	for k,v := range vm_total{
// 		fmt.Println("VM: ", k, ", Total Usage: ", v)
// 	}
// 	fmt.Println("")

// 	for len(vms_overused_queue) > 0 {
// 		key := vms_overused_queue[0]
// 		element := vm_container_map[key]

// 		// fmt.Println("Key: ", key)
// 		// for k,v := range element{
// 		// 	fmt.Println("Container: ", k, " Usage: ", v)
// 		// }

// 		if vm_total[key] > target {
// 			keys := make([]string, 0, len(element))
			
// 			for key_elem := range element {
// 				keys = append(keys, key_elem)
// 			}

// 			sort.SliceStable(keys, func(i, j int) bool{
// 				return element[keys[i]] >= element[keys[j]]
// 			})

// 			// fmt.Println("VM: ", key)
// 			// for _, k := range keys {
// 			// 	fmt.Println("Container: ", k)
// 			// }

// 			for _, key_elem := range keys {
// 				if (vm_total[key] <= target){
// 					break
// 				}
				
// 				isStillThere := true
// 				a := vm_index[key]
// 				initial_a := a


// 				for isStillThere {

// 					a = (a + 1)%total_vms
// 					if (a == initial_a){
// 						break
// 					}

// 					if (vm_total[vms[a]] + element[key_elem] < 1.00){
// 						vm_mapping[key_elem] = vms[a]
// 						vm_container_map[vms[a]][key_elem] = element[key_elem]
// 						vm_total[vms[a]] = vm_total[vms[a]] + element[key_elem]
// 						if (vm_total[vms[a]] > target){
// 							vms_overused_queue = append(vms_overused_queue, vms[a])
// 						}
// 						vm_total[key] = vm_total[key] - element[key_elem]
						
// 						isStillThere = false
// 						delete(vm_container_map[key], key_elem)
// 					}
					
// 				}

// 			}
// 		}
// 		vms_overused_queue = vms_overused_queue[1:]
// 	}

// 	fmt.Println("Printing VM's totals final:")
// 	for k,v := range vm_total{
// 		fmt.Println("VM: ", k, ", Total Usage: ", v)
// 	}
// 	fmt.Println("")
// 	fmt.Println("Target: ", target)
// 	fmt.Println("")
	
// 	return vm_mapping

// }

func scheduleMigration(vm_container_map map[string]map[string]float64, target float64) map[string]string {
	
	vm_total := make(map[string]float64)
	vm_mapping := make(map[string]string)
	total_vms := len(vm_container_map)
	vms := make([]string, 0, len(vm_container_map))

	vm_index := make(map[string]int)

	vms_overused_queue := make([]string, 0)

	count := 0
	for key, element := range vm_container_map{
		vm_total[key] = 0.0
		for _, cpu_perc := range element{
			vm_total[key] = vm_total[key] + cpu_perc
		}
		if (vm_total[key] > target){
			vms_overused_queue = append(vms_overused_queue, key)
		}
		vms = append(vms, key)
		vm_index[key] = count
		count++
	}

	fmt.Println("Printing VM's totals initial:")
	for k,v := range vm_total{
		fmt.Println("VM: ", k, ", Total Usage: ", v)
	}
	fmt.Println("")

	overused_redundancy := 0

	for len(vms_overused_queue) > 0 {
		key := vms_overused_queue[0]
		element := vm_container_map[key]

		// fmt.Println("Key: ", key)
		// for k,v := range element{
		// 	fmt.Println("Container: ", k, " Usage: ", v)
		// }

		if vm_total[key] > target {
			keys := make([]string, 0, len(element))
			
			for key_elem := range element {
				keys = append(keys, key_elem)
			}

			sort.SliceStable(keys, func(i, j int) bool{
				return element[keys[i]] >= element[keys[j]]
			})

			// fmt.Println("VM: ", key)
			// for _, k := range keys {
			// 	fmt.Println("Container: ", k)
			// }

			for _, key_elem := range keys {
				if (vm_total[key] <= target){
					break
				}
				
				isStillThere := true
				a := vm_index[key]
				initial_a := a


				for isStillThere {

					a = (a + 1)%total_vms
					if (a == initial_a){
						break
					}
					
					if ((vm_total[vms[a]] + element[key_elem] < 1.00) && ((vm_total[vms[a]] + element[key_elem]) < vm_total[key])){

						vm_mapping[key_elem] = vms[a]
						vm_container_map[vms[a]][key_elem] = element[key_elem]
						vm_total[vms[a]] = vm_total[vms[a]] + element[key_elem]
						if (vm_total[vms[a]] > target){
							vms_overused_queue = append(vms_overused_queue, vms[a])
						}
						vm_total[key] = vm_total[key] - element[key_elem]
						
						isStillThere = false
						delete(vm_container_map[key], key_elem)
					}	
				}
			}
		}
		
		vms_overused_queue = vms_overused_queue[1:]
		if (vm_total[key] > target){
			vms_overused_queue = append(vms_overused_queue, key)
			overused_redundancy++;
		} else{
			overused_redundancy = 0
		}

		if (overused_redundancy > len(vms_overused_queue)){
			break
		}

	}

	fmt.Println("Printing VM's totals final:")
	for k,v := range vm_total{
		fmt.Println("VM: ", k, ", Total Usage: ", v)
	}
	fmt.Println("")
	fmt.Println("Target: ", target)
	fmt.Println("")
	
	return vm_mapping

}

func main() {

		// Example 1
		// container_map := make(map[string]map[string]float64)
		// container_map["v1"] = make(map[string]float64)
		// container_map["v2"] = make(map[string]float64)
		// container_map["v3"] = make(map[string]float64)

		// container_map["v1"]["c1"] = 0.20
		// container_map["v1"]["c2"] = 0.05
		// container_map["v1"]["c3"] = 0.60
		// container_map["v1"]["c4"] = 0.05
		

		// container_map["v2"]["c5"] = 0.68

		// container_map["v3"]["c6"] = 0.25
		// container_map["v3"]["c7"] = 0.25

		// Example 2
		container_map := make(map[string]map[string]float64)
		container_map["v1"] = make(map[string]float64)
		container_map["v2"] = make(map[string]float64)
		container_map["v3"] = make(map[string]float64)

		container_map["v1"]["c1"] = 0.10
		container_map["v1"]["c2"] = 0.10
		container_map["v1"]["c3"] = 0.05
		container_map["v1"]["c4"] = 0.60
		container_map["v1"]["c5"] = 0.05
		

		container_map["v2"]["c6"] = 0.68
		container_map["v2"]["c7"] = 0.10

		container_map["v3"]["c6"] = 0.25
		container_map["v3"]["c7"] = 0.25
		container_map["v3"]["c8"] = 0.25

		target := 0.71

		// vm_map := scheduleMigration(container_map, target)
		vm_map := scheduleMigration(container_map, target)
		
		fmt.Println(vm_map)
		for k, v := range vm_map {
			fmt.Println(k, " Container scheduled to VM:", v)
		}
}