package scheduling

import (
	"github.com/draffensperger/golp"
	"github.com/grussorusso/serverledge/internal/node"
	"log"
	"time"
)

var numberOfInstances int
var functionPColdMap = make(map[string]int)

type fInfoWithClass struct {
	*classFunctionInfo
	class string
}

func getPColdIndex(name string) int {
	return numberOfInstances*4 + functionPColdMap[name]
}

func getPExecutionIndex(index int) int {
	return index * 4
}

func getPOffloadIndex(index int) int {
	return index*4 + 1
}

func getPDropIndex(index int) int {
	return index*4 + 2
}

func getShareIndex(index int) int {
	return index*4 + 3
}

// TODO separate pcold
func SolveProbabilities(m map[string]map[string]*classFunctionInfo) {
	if len(m) == 0 {
		return
	}

	list := make([]fInfoWithClass, 0)

	objectiveFunctionEntries := make([]float64, 0)
	memoryConstraintEntries := make([]golp.Entry, 0)
	cpuConstraintEntries := make([]golp.Entry, 0)

	classMap := make(map[string][]*classFunctionInfo)
	functionPdIndex := make(map[string]int)

	functionNumber := len(m)

	for _, submap := range m {
		for class, cFInfo := range submap {
			list = append(list, fInfoWithClass{cFInfo, class})

			classFunctionList, prs := classMap[class]
			if !prs {
				classFunctionList = make([]*classFunctionInfo, 1)
				classFunctionList[0] = cFInfo
			} else {
				classFunctionList = append(classFunctionList, cFInfo)
			}

			classMap[class] = classFunctionList
		}
	}

	numberOfInstances = len(list)

	index := 0
	for fName := range m {
		functionPColdMap[fName] = numberOfInstances*4 + index

		index++
	}

	//4 for every function, class pair and one pcold for each function
	lp := golp.NewLP(0, numberOfInstances*4+functionNumber)

	// 0 pe
	// 1 po
	// 2 pd
	// 3 x
	for i := range list {
		//Probability constraints
		//TODO needed if the sum is < 1?
		cFInfo := list[i]

		//Probability constraints
		lp.AddConstraintSparse([]golp.Entry{{getPExecutionIndex(i), 1.0}}, golp.LE, 1)
		lp.AddConstraintSparse([]golp.Entry{{getPOffloadIndex(i), 1.0}}, golp.LE, 1)
		lp.AddConstraintSparse([]golp.Entry{{getPDropIndex(i), 1.0}}, golp.LE, 1)

		//Sum of pe + pd + po = 1
		lp.AddConstraintSparse([]golp.Entry{{getPExecutionIndex(i), 1.0},
			{getPOffloadIndex(i), 1.0}, {getPDropIndex(i), 1.0}}, golp.EQ, 1)

		//Constraint for the scaling value
		//pe*time*arrival <= scale
		lp.AddConstraintSparse([]golp.Entry{{getPExecutionIndex(i), cFInfo.meanDuration[LOCAL] * cFInfo.arrivals},
			{getShareIndex(i), -1}}, golp.LE, 0)

		//Response time solution
		if Classes[cFInfo.class].MaximumResponseTime != -1 {
			lp.AddConstraintSparse([]golp.Entry{{getPExecutionIndex(i), cFInfo.meanDuration[LOCAL]},
				{getPOffloadIndex(i), cFInfo.offloadTime + cFInfo.meanDuration[OFFLOADED]},
				{getPColdIndex(list[i].name), cFInfo.initTime}},
				golp.LE, Classes[cFInfo.class].MaximumResponseTime)
		}

		log.Println(cFInfo.arrivals, Classes[cFInfo.class].Utility)
		objectiveFunctionEntries = append(objectiveFunctionEntries,
			[]float64{cFInfo.arrivals * Classes[cFInfo.class].Utility,
				cFInfo.arrivals * Classes[cFInfo.class].Utility,
				0,
				1}...)

		log.Println("CPU: ", cFInfo.cpu)
		memoryConstraintEntries = append(memoryConstraintEntries, []golp.Entry{{getShareIndex(i), float64(cFInfo.memory)}}...)

		//TODO functions can have 0 CPU demand?
		if cFInfo.cpu != 0 {
			cpuConstraintEntries = append(cpuConstraintEntries, []golp.Entry{{getShareIndex(i), cFInfo.cpu}}...)
		}

		functionPdIndex[cFInfo.name+cFInfo.class] = getPDropIndex(i)
	}

	//Class constraint
	for k, classList := range classMap {
		classConstraintEntries := make([]golp.Entry, 0)
		arrivalSum := 0.0

		for i := range classList {
			classConstraintEntries =
				append(classConstraintEntries, []golp.Entry{{functionPdIndex[classList[i].name+k], classList[i].arrivals}}...)

			arrivalSum += classList[i].arrivals
		}

		lp.AddConstraintSparse(classConstraintEntries, golp.LE, (1-Classes[k].CompletedPercentage)*arrivalSum)
	}

	for _, index := range functionPColdMap {
		lp.AddConstraintSparse([]golp.Entry{{index, 1.0}}, golp.LE, 1)

		objectiveFunctionEntries = append(objectiveFunctionEntries, 1)
	}

	if len(memoryConstraintEntries) > 0 {
		lp.AddConstraintSparse(memoryConstraintEntries, golp.LE, float64(node.Resources.MaxMemMB))
	}

	if len(cpuConstraintEntries) > 0 {
		lp.AddConstraintSparse(cpuConstraintEntries, golp.LE, node.Resources.MaxCPUs)
	}

	//Objective function
	lp.SetObjFn(objectiveFunctionEntries)
	lp.SetMaximize()

	start := time.Now()
	sol := lp.Solve()
	elapsed := time.Since(start)

	vars := lp.Variables()

	for i := range list {
		fInfo := list[i]

		fInfo.probExecute = vars[i*5]
		fInfo.probOffload = vars[i*5+1]
		fInfo.probDrop = vars[i*5+2]
	}

	log.Println(lp.WriteToString())
	log.Printf("Resolution took %s", elapsed)
	log.Println("Var: ", vars)
	log.Println("Sol type: ", sol)
	log.Println("Optimum: ", lp.Objective())
}

/*
func Solve() {
	lp := golp.NewLP(0, 5)

	lp.AddConstraint([]float64{1, 1, 1, 0, 0}, golp.EQ, 1)

	lp.AddConstraint([]float64{1.0, 0, 0, 0, 0}, golp.LE, 1)
	lp.AddConstraint([]float64{0, 1.0, 0, 0, 0}, golp.LE, 1)
	lp.AddConstraint([]float64{0, 0, 1.0, 0, 0}, golp.LE, 1)
	lp.AddConstraint([]float64{0, 0, 0, 1.0, 0}, golp.LE, 1)

	lp.AddConstraintSparse([]golp.Entry{{0, 1.0}}, golp.LE, 1)

	lp.AddConstraint([]float64{0, 0, 0, 0, memory}, golp.LE, maxMemory)
	lp.AddConstraint([]float64{0, 0, 0, 0, cpu}, golp.LE, maxCpu)

	lp.AddConstraint([]float64{serviceTime * arrivals, 0, 0, 0, -1}, golp.LE, 0)

	lp.AddConstraint([]float64{serviceTime, cloudServiceTime + offloadingTime, 0, initTime, 0}, golp.LE, maxServiceTime)

	lp.AddConstraint([]float64{0, 0, arrivals, 0, 0}, golp.LE, (1-minExecuted)*arrivals)

	lp.SetObjFn([]float64{arrivals * utility, arrivals * utility, 0, 1, 0})
	lp.SetMaximize()

	start := time.Now()
	lp.Solve()
	elapsed := time.Since(start)

	log.Printf("Resolution took %s", elapsed)

	vars := lp.Variables()
	log.Printf("Rows: %d Cols: %d\n", lp.NumRows(), lp.NumCols())
	log.Printf("PE %.3f\n", vars[0])
	log.Printf("PO  %.3f\n", vars[1])
	log.Printf("PD %.3f\n", vars[2])
	log.Printf("PC  %.3f\n", vars[3])
	log.Printf("X  %.3f\n", vars[4])
	log.Printf("For %.2f\n", lp.Objective())
	log.Println(vars)
	log.Println(lp.WriteToString())
}

*/
