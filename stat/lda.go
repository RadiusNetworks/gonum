// Package stat provides generalized statistical functions
package stat

import (
	"fmt"
	"math"
	"math/cmplx"
	"sort"

	"gonum.org/v1/gonum/mat"
)

// LD is a type for computing and extracting the linear discriminant analysis of a
// matrix. The results of the linear discriminant analysis are only valid
// if the call to LinearDiscriminant was successful.
type LD struct {
	n, p  int //n = row, p = col
	k     int
	ct    []float64  //Constant term of discriminant function of each class
	mu    *mat.Dense //Mean vectors of each class
	svd   *mat.SVD
	ok    bool
	eigen mat.Eigen //Eigen values of common variance matrix
}

// LinearDiscriminant performs a linear discriminant analysis on the
// matrix of the input data which is represented as an n×p matrix x where each
// row is an observation and each column is a variable
//
// LinearDiscriminant returns whether the analysis was successful
//
// @param x is the training samples
// @param y is the training labels in [0,k)
// where k is the number of classes
// @retun ok returns if whether the analysis was successful
func (ld *LD) LinearDiscriminant(x mat.Matrix, y []int) (ok bool) {
	ld.n, ld.p = x.Dims()
	fmt.Printf("This is the matrix: %v \n", x)
	fmt.Printf("This is the array: %v \n", y)
	fmt.Printf("x dims: %v, %v \n", ld.n, ld.p)
	if y != nil && len(y) != ld.n {
		panic("The sizes of X and Y don't match")
	}
	var labels []int
	var found bool
	//Find unique labels
	for i := 0; i < len(y); i++ {
		found = false
		for j := 0; j < len(labels); j++ {
			if y[i] == labels[j] {
				found = true
				break
			}
		}
		if !found {
			labels = append(labels, y[i])
		}

	}
	//Create a new array with labels and go through the array of y values and if
	//it doesnt exist then add it to the new array
	sort.Ints(labels)
	fmt.Printf("Sorted list of labels: %v \n", labels)

	if labels[0] != 0 {
		panic("Label does not start from zero")
	}
	for i := 0; i < len(labels); i++ {
		if labels[i] < 0 {
			panic("Negative class label")
		}
		if i > 0 && labels[i]-labels[i-1] > 1 {
			panic("Missing class")
		}
	}
	//Tol is a tolerence to decide if a covariance matrix is singular (det is zero)
	//Tol will reject variables whose variance is less than tol
	var tol float64 = 1E-4
	//k is the number of classes
	ld.k = len(labels)
	fmt.Printf("this is k and ld.n: %v, %v \n", ld.k, ld.n)
	if ld.k < 2 {
		panic("Only one class.")
	}
	if tol < 0.0 {
		panic("Invalid tol")
	}
	if ld.n <= ld.k {
		panic("Sample size is too small")
	}

	//Number of instances in each class
	ni := make([]int, ld.k)

	//Common mean vector
	var colmean []float64
	for i := 0; i < ld.p; i++ {
		var col []float64 = mat.Col(nil, i, x)
		var sum float64 = 0
		for _, value := range col {
			sum += value
		}
		colmean = append(colmean, sum/float64(ld.n))
	}
	fmt.Printf("this is the array of means %v \n", colmean)

	//C is a matrix of zeros with dimensions: ld.p x ld.p
	C := mat.NewDense(ld.p, ld.p, make([]float64, ld.p*ld.p, ld.p*ld.p))
	fmt.Printf("this is the zero matrix: %v \n", C)

	//Class mean vectors
	//mu is a matrix with dimensions: k x ld.p
	ld.mu = mat.NewDense(ld.k, ld.p, make([]float64, ld.k*ld.p, ld.k*ld.p))
	for i := 0; i < ld.n; i++ {
		ni[y[i]] = ni[y[i]] + 1
		for j := 0; j < ld.p; j++ {
			ld.mu.Set(y[i], j, ((ld.mu.At(y[i], j)) + (x.At(i, j))))
		}
	}
	for i := 0; i < ld.k; i++ {
		for j := 0; j < ld.p; j++ {
			ld.mu.Set(i, j, ((ld.mu.At(i, j)) / (float64)(ni[i])))
		}
	}

	//priori is the priori probability of each class
	priori := make([]float64, ld.k)
	for i := 0; i < ld.k; i++ {
		priori[i] = (float64)(ni[i] / ld.n)
	}

	//ct is the constant term of discriminant function of each class
	ld.ct = make([]float64, ld.k)
	for i := 0; i < ld.k; i++ {
		ld.ct[i] = math.Log(priori[i])
	}

	for i := 0; i < ld.n; i++ {
		for j := 0; j < ld.p; j++ {
			for l := 0; l <= j; l++ {
				C.Set(j, l, (C.At(j, l) + ((x.At(i, j) - colmean[j]) * (x.At(i, l) - colmean[l]))))
			}
		}
	}

	tol = tol * tol

	for j := 0; j < ld.p; j++ {
		for l := 0; l <= j; l++ {
			C.Set(j, l, ((C.At(j, l)) / (float64)(ld.n-ld.k)))
			C.Set(l, j, C.At(j, l))
		}
		if C.At(j, j) < tol {
			panic("Covarience matrix (variable %d) is close to singular")
		}
	}

	fmt.Printf("this is the code varience %v \n", C)

	//Factorize returns whether the decomposition succeeded
	//If the decomposition failed, methods that require a successful factorization will panic
	ld.eigen.Factorize(C, false, true)
	fmt.Printf("this is the eigen value %v \n", ld.eigen)
	return true
}

// Transform performs a transformation on the
// matrix of the input data which is represented as an ld.n × p matrix x
//
// Transform returns the transformed matrix
//
// @param x is the matrix
// @retun result matrix
func (ld *LD) Transform(x mat.Matrix) *mat.Dense {
	_, p := ld.eigen.Vectors().Dims()
	result := mat.NewDense(ld.n, p, make([]float64, ld.n*p, ld.n*p))
	result.Mul(x, ld.eigen.Vectors())
	return result
}

func (ld *LD) Predict(x []float64) int {
	if len(x) != ld.p {
		panic("Invalid imput vector size")
	}
	var y int = 0
	var max float64 = math.Inf(-1)
	d := make([]float64, ld.p)
	ux := make([]float64, ld.p)
	for i := 0; i < ld.k; i++ {
		for j := 0; j < ld.p; j++ {
			d[j] = x[j] - ld.mu.At(i, j)
		}
		var f float64 = 0.0
		evals := make([]complex128, ld.p)
		ld.eigen.Values(evals)
		for j := 0; j < ld.p; j++ {
			f += ux[j] * ux[j] / cmplx.Abs(evals[j])
		}
		f = ld.ct[i] - 0.5*f
		if max < f {
			max = f
			y = i
		}
	}
	return y

}
