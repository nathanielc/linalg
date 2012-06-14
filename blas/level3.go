
// Copyright (c) Harri Rautila, 2012

// This file is part of go.opt/linalg package. It is free software, distributed
// under the terms of GNU Lesser General Public License Version 3, or any later
// version. See the COPYING tile included in this archive.

package blas

import (
	"linalg"
	"matrix"
	"errors"
	"math"
	"math/cmplx"
)

/*
 General matrix-matrix product. (L3)
 
 PURPOSE
 Computes 
  C := alpha*A*B + beta*C     if transA = PNoTrans   and transB = PNoTrans.
  C := alpha*A^T*B + beta*C   if transA = PTrans     and transB = PNoTrans.
  C := alpha*A^H*B + beta*C   if transA = PConjTrans and transB = PNoTrans.
  C := alpha*A*B^T + beta*C   if transA = PNoTrans   and transB = PTrans.
  C := alpha*A^T*B^T + beta*C if transA = PTrans     and transB = PTrans.
  C := alpha*A^H*B^T + beta*C if transA = PConjTrans and transB = PTrans.
  C := alpha*A*B^H + beta*C   if transA = PNoTrans   and transB = PConjTrans.
  C := alpha*A^T*B^H + beta*C if transA = PTrans     and transB = PConjTrans.
  C := alpha*A^H*B^H + beta*C if transA = PConjTrans and transB = PConjTrans.

 The number of rows of the matrix product is m.  The number of  columns is n.
 The inner dimension is k.  If k=0, this reduces  to C := beta*C.

 ARGUMENTS
  A         float or complex matrix, m*k
  B         float or complex matrix, k*n
  C         float or complex matrix, m*n
  alpha     number (float or complex singleton matrix)
  beta      number (float or complex singleton matrix)

 OPTIONS
  transA    PNoTrans, PTrans or PConjTrans
  transB    PNoTrans, PTrans or PConjTrans
  m         integer.  If negative, the default value is used. The default value is
            m = A.Rows of if transA != PNoTrans m = A.Cols.
  n         integer.  If negative, the default value is used. The default value is
            n = (transB == PNoTrans) ? B.Cols : B.Rows.
  k         integer.  If negative, the default value is used. The default value is
            k=A.Cols or if transA != PNoTrans) k = A.Rows, transA=PNoTrans.
            If the default value is used it should also be equal to
            (transB == PNoTrans) ? B.Rows : B.Cols.
  ldA       nonnegative integer.  ldA >= max(1,m) of if transA != NoTrans max(1,k).
            If zero, the default value is used.
  ldB       nonnegative integer.  ldB >= max(1,k) or if transB != NoTrans max(1,n).
            If zero, the default value is used.
  ldC       nonnegative integer.  ldC >= max(1,m).
            If zero, the default value is used.
  offsetA   nonnegative integer
  offsetB   nonnegative integer
  offsetC   nonnegative integer;
*/
func Gemm(A, B, C, alpha, beta matrix.Matrix, opts ...linalg.Opt) (err error) {

	params, e := linalg.GetParameters(opts...)
	if e != nil {
		err = e
		return
	}
	ind := linalg.GetIndexOpts(opts...)
	err = check_level3_func(ind, fgemm, A, B, C, params)
	if err != nil {
		return
	}
	if ind.M == 0 || ind.N == 0 {
		return
	}
	if ! matrix.EqualTypes(A, B, C) {
		return errors.New("Parameters not of same type")
	}
	switch A.(type) {
	case *matrix.FloatMatrix:
		Aa := A.FloatArray()
		Ba := B.FloatArray()
		Ca := C.FloatArray()
		aval := 1.0
		if alpha != nil {
			aval = alpha.FloatValue()
		}
		bval := 0.0
		if beta != nil {
			bval = beta.FloatValue()
		}
		if math.IsNaN(aval) || math.IsNaN(bval) {
			return errors.New("alpha or beta not a number")
		}
		transB := linalg.ParamString(params.TransB)
		transA := linalg.ParamString(params.TransA)
		dgemm(transA, transB, ind.M, ind.N, ind.K, aval,
			Aa[ind.OffsetA:], ind.LDa, Ba[ind.OffsetB:], ind.LDb, bval,
			Ca[ind.OffsetC:], ind.LDc)

	case *matrix.ComplexMatrix:
		Aa := A.ComplexArray()
		Ba := B.ComplexArray()
		Ca := C.ComplexArray()
		aval := complex(1.0, 0.0)
		if alpha != nil {
			aval = alpha.ComplexValue()
			if cmplx.IsNaN(aval) {
				aval = complex(alpha.FloatValue(), 0.0)
			} else {
				return errors.New("alpha not a number")
			}
		}
		bval := complex(0.0, 0.0)
		if beta != nil {
			bval = beta.ComplexValue()
			if cmplx.IsNaN(bval) {
				aval = complex(beta.FloatValue(), 0.0)
			} else {
				return errors.New("beta not a number")
			}
		}
		transB := linalg.ParamString(params.TransB)
		transA := linalg.ParamString(params.TransA)
		zgemm(transA, transB, ind.M, ind.N, ind.K, aval,
			Aa[ind.OffsetA:], ind.LDa, Ba[ind.OffsetB:], ind.LDb, bval,
			Ca[ind.OffsetC:], ind.LDc)
	default:
		return errors.New("Unknown type, not implemented")
	}
	return
}

/*
 Matrix-matrix product where one matrix is symmetric. (L3)

 Computes
  C := alpha*A*B + beta*C, if side is PLeft 
  C := alpha*B*A + beta*C, if side is PRight

 C is m by n and A is real symmetric. 

 ARGUMENTS
  A         float or complex matrix
  B         float or complex matrix.
  C         float m*n matrix.
  alpha     number (float).
  beta      number (float). 

 OPTIONS
  side      PLeft or PRight'
  uplo      PLower or PUpper
  m         integer.  If negative, the default value is used.
            If the default value is used and side = PLeft, then m
            must be equal to A.Rows and A.Cols.
  n         integer.  If negative, the default value is used.
            If the default value is used and side = PRight, then 
            must be equal to A.Rows and A.Cols.
  ldA       nonnegative integer.
            ldA >= max(1, m) or if side == PRight ldA >= max(1, n).
		    If zero, the default value is used.
  ldB       nonnegative integer.
            ldB >= max(1,n) or if side == PRight ldB >= max(1, m).
            If zero, the default value is used.
  ldC       nonnegative integer.  ldC >= max(1,m). If zero, the default value is used.
  offsetA   nonnegative integer
  offsetB   nonnegative integer
  offsetC   nonnegative integer

*/
func Symm(A, B, C, alpha, beta matrix.Matrix, opts ...linalg.Opt) (err error) {

	params, e := linalg.GetParameters(opts...)
	if e != nil {
		err = e
		return
	}
	ind := linalg.GetIndexOpts(opts...)
	err = check_level3_func(ind, fsymm, A, B, C, params)
	if err != nil {
		return
	}
	if ind.M == 0 || ind.N == 0 {
		return
	}
	if ! matrix.EqualTypes(A, B, C) {
		return errors.New("Parameters not of same type")
	}
	switch A.(type) {
	case *matrix.FloatMatrix:
		Aa := A.FloatArray()
		Ba := B.FloatArray()
		Ca := C.FloatArray()
		aval := 1.0
		if alpha != nil {
			aval = alpha.FloatValue()
		}
		bval := 0.0
		if beta != nil {
			bval = beta.FloatValue()
		}
		if math.IsNaN(aval) || math.IsNaN(bval) {
			return errors.New("alpha or beta not a number")
		}
		uplo := linalg.ParamString(params.Uplo)
		side := linalg.ParamString(params.Side)
		dsymm(side, uplo, ind.M, ind.N, aval, Aa[ind.OffsetA:], ind.LDa,
			Ba[ind.OffsetB:], ind.LDb, bval, Ca[ind.OffsetC:], ind.LDc)
	case *matrix.ComplexMatrix:
		Aa := A.ComplexArray()
		Ba := B.ComplexArray()
		Ca := C.ComplexArray()
		aval := complex(1.0, 0.0)
		if alpha != nil {
			aval = alpha.ComplexValue()
			if cmplx.IsNaN(aval) {
				aval = complex(alpha.FloatValue(), 0.0)
			} else {
				return errors.New("alpha not a number")
			}
		}
		bval := complex(0.0, 0.0)
		if beta != nil {
			bval = beta.ComplexValue()
			if cmplx.IsNaN(bval) {
				bval = complex(beta.FloatValue(), 0.0)
			} else {
				return errors.New("beta not a number")
			}
		}
		uplo := linalg.ParamString(params.Uplo)
		side := linalg.ParamString(params.Side)
		zhemm(side, uplo, ind.M, ind.N, aval, Aa[ind.OffsetA:], ind.LDa,
			Ba[ind.OffsetB:], ind.LDb, bval, Ca[ind.OffsetC:], ind.LDc)
	default:
		return errors.New("Unknown type, not implemented")
	}

	return
}

func Hemm(A, B, C, alpha, beta matrix.Matrix, opts ...linalg.Opt) (err error) {
	err = Symm(A, B, C, alpha, beta, opts...)
	return 
}

/*
 Rank-k update of symmetric matrix. (L3)

 Syrk(A, C, uplo=PLower, trans=PNoTrans, alpha=1.0, beta=0.0, n=-1,
 k=-1, ldA=max(1,A.Rows), ldC=max(1,C.Rows), offsetA=0, offsetB=0)

 PURPOSE   
  C := alpha*A*A^T + beta*C, if trans is PNoTrans
  C := alpha*A^T*A + beta*C, if trans is PTrans
  C := alpha*A^H*A + beta*C, if trans is PConjTrans
  
 C is symmetric (real or complex) of order n. 
 The inner dimension of the matrix product is k.  If k=0 this is
 interpreted as C := beta*C.

 ARGUMENTS
  A         float or complex n*k matrix 
  C         float or complex n*n matrix.  Must have the same type as A.
  alpha     number (float or complex singleton matrix).  Complex alpha is only
            allowed if A is complex.
  beta      number (float or complex singleton matrix).  Complex beta is only
            allowed if A is complex.

 OPTIONS
  uplo      PLower or PUpper
  trans     PNoTrans, PTrans or PConjTrans
  n         integer.  If negative, the default value is used.
            The default value is  n = A.Rows of if trans != PNoTrans) n = A.Cols.
  k         integer.  If negative, the default value is used.
            The default value is k = A.Cols or if trans != PNoTrans k = A.Rows.
  ldA       nonnegative integer. ldA >= max(1, n) or if trans != PNoTrans max(1, k).
            If zero, the default value [max(1, A.Rows)] is used.
  ldC       nonnegative integer.  ldC >= max(1,n).  If zero, the default value is used.
  offsetA   nonnegative integer
  offsetC   nonnegative integer;
*/
func Syrk(A, C, alpha, beta matrix.Matrix, opts ...linalg.Opt) (err error) {

	params, e := linalg.GetParameters(opts...)
	if e != nil {
		err = e
		return
	}
	ind := linalg.GetIndexOpts(opts...)
	err = check_level3_func(ind, fsyrk, A, nil, C, params)
	if e != nil || err != nil {
		return
	}
	if ! matrix.EqualTypes(A, C) {
		return errors.New("Parameters not of same type")
	}
	switch A.(type) {
	case *matrix.FloatMatrix:
		Aa := A.FloatArray()
		Ca := C.FloatArray()
		aval := 1.0
		if alpha != nil {
			aval = alpha.FloatValue()
		}
		bval := 0.0
		if beta != nil {
			bval = beta.FloatValue()
		}
		if math.IsNaN(aval) || math.IsNaN(bval) {
			return errors.New("alpha or beta not a number")
		}
		uplo := linalg.ParamString(params.Uplo)
		trans := linalg.ParamString(params.Trans)
		dsyrk(uplo, trans, ind.N, ind.K, aval, Aa[ind.OffsetA:], ind.LDa, bval,
			Ca[ind.OffsetC:], ind.LDc)
	case *matrix.ComplexMatrix:
		Aa := A.ComplexArray()
		Ca := C.ComplexArray()
		aval := complex(1.0, 0.0)
		if alpha != nil {
			aval = alpha.ComplexValue()
			if cmplx.IsNaN(aval) {
				aval = complex(alpha.FloatValue(), 0.0)
			} else {
				return errors.New("alpha not a number")
			}
		}
		bval := complex(0.0, 0.0)
		if beta != nil {
			bval = beta.ComplexValue()
			if cmplx.IsNaN(bval) {
				bval = complex(beta.FloatValue(), 0.0)
			} else {
				return errors.New("beta not a number")
			}
		}
		uplo := linalg.ParamString(params.Uplo)
		trans := linalg.ParamString(params.Trans)
		zsyrk(uplo, trans, ind.N, ind.K, aval, Aa[ind.OffsetA:], ind.LDa, bval,
			Ca[ind.OffsetC:], ind.LDc)
	default:
		return errors.New("Unknown type, not implemented")
	}
		
	return
}

/*
 Rank-k update of symmetric matrix. (L3)

 Herk(A, C, alpha=1.0, beta=0.0, uplo=PLower, trans=PNoTrans,  n=-1, 
 k=-1, ldA=max(1,A.Rows), ldC=max(1,C.Rows), offsetA=0, offsetB=0)

 Computes
  C := alpha*A*A^T + beta*C, if trans is PNoTrans
  C := alpha*A^T*A + beta*C, if trans is PTrans

 C is symmetric (real or complex) of order n. The inner dimension of the matrix
 product is k.  If k=0 this is interpreted as C := beta*C.

 ARGUMENTS
  A         float or complex matrix.
  C         float or complex matrix.  Must have the same type as A.
  alpha     number (float or complex singleton matrix).  Complex alpha is only
            allowed if A is complex.
  beta      number (float or complex singleton matrix).  Complex beta is only
            allowed if A is complex.

 OPTIONS
  uplo      PLower or PUpper
  trans     PNoTrans or PTrans
  n         integer.  If negative, the default value is used.
            The default value is n = A.Rows or if trans == PNoTrans n = A.Cols.
  k         integer.  If negative, the default value is used.
            The default value is k =  A.Cols, or if trans == PNoTrans k = A.Rows.
  ldA       nonnegative integer.
            ldA >= max(1,n) or if trans != PNoTrans ldA >= max(1,k).
            If zero, the default value is used.
  ldC       nonnegative integer.  ldC >= max(1,n).
            If zero, the default value is used.
  offsetA   nonnegative integer
  offsetC   nonnegative integer;
*/
func Herk(A, C, alpha, beta matrix.Matrix, opts ...linalg.Opt) (err error) {

	params, e := linalg.GetParameters(opts...)
	if e != nil {
		err = e
		return
	}
	ind := linalg.GetIndexOpts(opts...)
	err = check_level3_func(ind, fsyrk, A, nil, C, params)
	if e != nil || err != nil {
		return
	}
	if ! matrix.EqualTypes(A, C) {
		return errors.New("Parameters not of same type")
	}
	switch A.(type) {
	case *matrix.FloatMatrix:
		Aa := A.FloatArray()
		Ca := C.FloatArray()
		aval := 1.0
		if alpha != nil {
			aval = alpha.FloatValue()
		}
		bval := 0.0
		if beta != nil {
			bval = beta.FloatValue()
		}
		if math.IsNaN(aval) || math.IsNaN(bval) {
			return errors.New("alpha or beta not a number")
		}
		uplo := linalg.ParamString(params.Uplo)
		trans := linalg.ParamString(params.Trans)
		dsyrk(uplo, trans, ind.N, ind.K, aval, Aa[ind.OffsetA:], ind.LDa, bval,
			Ca[ind.OffsetC:], ind.LDc)
	case *matrix.ComplexMatrix:
		Aa := A.ComplexArray()
		Ca := C.ComplexArray()
		aval := complex(1.0, 0.0)
		if alpha != nil {
			aval = alpha.ComplexValue()
			if cmplx.IsNaN(aval) {
				aval = complex(alpha.FloatValue(), 0.0)
			} else {
				return errors.New("alpha not a real or complex number")
			}
		}
		bval := 0.0
		if beta != nil {
			bval = beta.FloatValue()
			if math.IsNaN(bval) {
				return errors.New("beta not a real number")
			}
		}
		uplo := linalg.ParamString(params.Uplo)
		trans := linalg.ParamString(params.Trans)
		zherk(uplo, trans, ind.N, ind.K, aval, Aa[ind.OffsetA:], ind.LDa, bval,
			Ca[ind.OffsetC:], ind.LDc)
	default:
		return errors.New("Unknown type, not implemented")
	}
		
	return
}

/*
 Rank-2k update of symmetric matrix. (L3)

 syr2k(A, B, C, uplo=PLower, trans=PNoTrnas, alpha=1.0, beta=0.0, n=-1,
 k=-1, ldA=max(1,A.Rows), ldB=max(1,B.Rows), 
 ldC=max(1,C.Rows), offsetA=0, offsetB=0, offsetC=0)
 
 PURPOSE
  C := alpha*(A*B^T + B*A^T) + beta*C, if trans is NoTrans
  C := alpha*(A^T*B + B^T*A) + beta*C, if trans is Trans

 C is symmetric (real or complex) of order n. The inner dimension of the matrix
 product is k.  If k=0 this is interpreted as C := beta*C.


 ARGUMENTS
  A         float or complex matrix
  B         float or complex matrix.  Must have the same type as A.
  C         float or complex matrix.  Must have the same type as A.
  alpha     number (int, float or complex).  Complex alpha is only
            allowed if A is complex.
  beta      number (int, float or complex).  Complex beta is only
            allowed if A is complex.

 OPTIONS
  uplo      PLower or PUpper
  trans     PNoTrans, PTrans or PConjTrans (PConjTrans is only allowed when in the real
            case and means the same as PTrans)
  n         integer.  If negative, the default value is used.
            The default value is n = A.Rows or trans != PNoTrans n = A.Cols
            If the default value is used, it should be equal to B.Rows or 
            if trans != PNoTrans then B.Cols.
  k         integer.  If negative, the default value is used.
            The default value is  k = A.Cols or if trans != PNoTrans then k = A.Rows
            If the default value is used, it should be equal to B.Cols or if
            trans != PNoTrans then equal to B.Rows.
  ldA       nonnegative integer.  ldA >= max(1,n) or if trans != PNoTrans ldA >= max(1,k).
            If zero, the default value is used.
  ldB       nonnegative integer.
            ldB >= max(1,n) or if trans != PNoTrans then ldB >= max(1,k).
            If zero, the default value is used.
  ldC       nonnegative integer.  ldC >= max(1,n).
            If zero, the default value is used.
  offsetA   nonnegative integer
  offsetB   nonnegative integer
  offsetC   nonnegative integer
 
 */
func Syr2k(A, B, C, alpha, beta matrix.Matrix, opts ...linalg.Opt) (err error) {

	params, e := linalg.GetParameters(opts...)
	if e != nil {
		err = e
		return
	}
	ind := linalg.GetIndexOpts(opts...)
	err = check_level3_func(ind, fsyr2k, A, B, C, params)
	if err != nil {
		return
	}
	if ! matrix.EqualTypes(A, B, C) {
		return errors.New("Parameters not of same type")
	}
	switch A.(type) {
	case *matrix.FloatMatrix:
		Aa := A.FloatArray()
		Ba := B.FloatArray()
		Ca := C.FloatArray()
		aval := 1.0
		if alpha != nil {
			aval = alpha.FloatValue()
		}
		bval := 0.0
		if beta != nil {
			bval = beta.FloatValue()
		}
		if math.IsNaN(aval) || math.IsNaN(bval) {
			return errors.New("alpha or beta not a number")
		}
		uplo := linalg.ParamString(params.Uplo)
		trans := linalg.ParamString(params.Trans)
		dsyr2k(uplo, trans,	ind.N, ind.K, aval, Aa[ind.OffsetA:], ind.LDa,
			Ba[ind.OffsetB:], ind.LDb, bval, Ca[ind.OffsetC:], ind.LDc)

	case *matrix.ComplexMatrix:
		Aa := A.ComplexArray()
		Ba := B.ComplexArray()
		Ca := C.ComplexArray()
		aval := complex(1.0, 0.0)
		if alpha != nil {
			aval = alpha.ComplexValue()
			if cmplx.IsNaN(aval) {
				if ! math.IsNaN(alpha.FloatValue()) {
					aval = complex(alpha.FloatValue(), 0.0)
				} else {
					return errors.New("alpha not a real or complex number")
				}
			}
		}
		bval := complex(0.0, 0.0)
		if beta != nil {
			bval = beta.ComplexValue()
			if cmplx.IsNaN(bval) {
				if ! math.IsNaN(beta.FloatValue()) {
					bval = complex(beta.FloatValue(), 0.0)
				} else {
					return errors.New("beta not a real or complex number")
				}
			}
		}
		uplo := linalg.ParamString(params.Uplo)
		trans := linalg.ParamString(params.Trans)
		zsyr2k(uplo, trans,	ind.N, ind.K, aval, Aa[ind.OffsetA:], ind.LDa,
			Ba[ind.OffsetB:], ind.LDb, bval, Ca[ind.OffsetC:], ind.LDc)
	default:
		return errors.New("Unknown type, not implemented")
	}
	return
}

/*
 Rank-2k update of symmetric matrix. (L3)

 Her2k(A, B, C,  alpha=1.0, beta=0.0, uplo=PLower, trans=PNoTrans, n=-1,
 k=-1, ldA=max(1,A.Rows), ldB=max(1,B.Rows), 
 ldC=max(1,C.Rows), offsetA=0, offsetB=0, offsetC=0)

 PURPOSE
  C := alpha*(A*B^T + B*A^T) + beta*C, if trans is PNoTrans
  C := alpha*(A^T*B + B^T*A) + beta*C, if trans is PTrans

 C is symmetric (real or complex) of order n. The inner dimension of the matrix
 product is k.  If k=0 this is interpreted as C := beta*C.

 ARGUMENTS
  A         float or complex matrix
  B         float or complex matrix.  Must have the same type as A.
  C         float or complex matrix.  Must have the same type as A.
  alpha     number (int, float or complex).  Complex alpha is only
            allowed if A is complex.
  beta      number (int, float or complex).  Complex beta is only
            allowed if A is complex.

 OPTIONS
  uplo      PLower or PUpper
  trans     PNoTrans, PTrans or PConjTrans (PConjTrans is only allowed when in the real
            case and means the same as PTrans)
  n         integer.  If negative, the default value is used.
            The default value is n = A.Rows or trans != PNoTrans n = A.Cols
            If the default value is used, it should be equal to B.Rows or 
            if trans != PNoTrans then B.Cols.
  k         integer.  If negative, the default value is used.
            The default value is  k = A.Cols or if trans != PNoTrans then k = A.Rows
            If the default value is used, it should be equal to B.Cols or if
            trans != PNoTrans then equal to B.Rows.
  ldA       nonnegative integer.  ldA >= max(1,n) or if trans != PNoTrans ldA >= max(1,k).
            If zero, the default value is used.
  ldB       nonnegative integer.
            ldB >= max(1,n) or if trans != PNoTrans then ldB >= max(1,k).
            If zero, the default value is used.
  ldC       nonnegative integer.  ldC >= max(1,n).
            If zero, the default value is used.
  offsetA   nonnegative integer
  offsetB   nonnegative integer
  offsetC   nonnegative integer
 */
func Her2k(A, B, C, alpha, beta matrix.Matrix, opts ...linalg.Opt) (err error) {

	params, e := linalg.GetParameters(opts...)
	if e != nil {
		err = e
		return
	}
	ind := linalg.GetIndexOpts(opts...)
	err = check_level3_func(ind, fsyr2k, A, B, C, params)
	if err != nil {
		return
	}
	if ! matrix.EqualTypes(A, B, C) {
		return errors.New("Parameters not of same type")
	}
	switch A.(type) {
	case *matrix.FloatMatrix:
		Aa := A.FloatArray()
		Ba := B.FloatArray()
		Ca := C.FloatArray()
		aval := 1.0
		if alpha != nil {
			aval = alpha.FloatValue()
		}
		bval := 0.0
		if beta != nil {
			bval = beta.FloatValue()
		}
		if math.IsNaN(aval) || math.IsNaN(bval) {
			return errors.New("alpha or beta not a number")
		}
		uplo := linalg.ParamString(params.Uplo)
		trans := linalg.ParamString(params.Trans)
		dsyr2k(uplo, trans,	ind.N, ind.K, aval, Aa[ind.OffsetA:], ind.LDa,
			Ba[ind.OffsetB:], ind.LDb, bval, Ca[ind.OffsetC:], ind.LDc)

	case *matrix.ComplexMatrix:
		Aa := A.ComplexArray()
		Ba := B.ComplexArray()
		Ca := C.ComplexArray()
		aval := complex(1.0, 0.0)
		if alpha != nil {
			aval = alpha.ComplexValue()
			if cmplx.IsNaN(aval) {
				aval = complex(alpha.FloatValue(), 0.0)
			} else {
				return errors.New("alpha not a number")
			}
		}
		bval := 0.0
		if beta != nil {
			bval = beta.FloatValue()
			if math.IsNaN(bval) {
				return errors.New("beta not a real number")
			}
		}
		uplo := linalg.ParamString(params.Uplo)
		trans := linalg.ParamString(params.Trans)
		zher2k(uplo, trans,	ind.N, ind.K, aval, Aa[ind.OffsetA:], ind.LDa,
			Ba[ind.OffsetB:], ind.LDb, bval, Ca[ind.OffsetC:], ind.LDc)
	default:
		return errors.New("Unknown type, not implemented")
	}
	return
}

/*
 Matrix-matrix product where one matrix is triangular. (L3)

 Trmm(A, B, alpha=1.0,side=PLeft, uplo=PLower, transA=PNoTrans, diag=PNonUnit, 
 m=-1, n=-1, ldA=max(1,A.Rows), ldB=max(1,B.Rows), offsetA=0, offsetB=0)

 Computes
  B := alpha*A*B   if transA is PNoTrans   and side = PLeft
  B := alpha*B*A   if transA is PNoTrans   and side = PRight
  B := alpha*A^T*B if transA is PTrans     and side = PLeft
  B := alpha*B*A^T if transA is PTrans     and side = PRight
  B := alpha*A^H*B if transA is PConjTrans and side = PLeft
  B := alpha*B*A^H if transA is PConjTrans and side = PRight

 B is m by n and A is triangular.
 
 ARGUMENTS
  A         float or complex matrix
  B         float or complex matrix.  Must have the same type as A.
  alpha     number (float or complex singleton matrix).  Complex alpha is only
            allowed if A is complex.

 OPTIONS
  side      PLeft or PRight
  uplo      PLower or PUpper
  transA    PNoTrans or PTrans
  diag      PNonUnit or PUnit
  m         integer.  If negative, the default value is used.
            The default value is m = A.Rows or if side == PRight m = B.Rows
            If the default value is used and side is PLeft, m must be equal to A.Cols.
  n         integer.  If negative, the default value is used.
            The default value is n = B.Cols or if side )= PRight n = A.Rows.
            If the default value is used and side is PRight, n must be equal to A.Cols.
  ldA       nonnegative integer.
            ldA >= max(1,m) of if  side == PRight lda >= max(1,n).
            If zero, the default value is used. 
  ldB       nonnegative integer.  ldB >= max(1,m).
            If zero, the default value is used.
  offsetA   nonnegative integer
  offsetB   nonnegative integer
 */
func Trmm(A, B, alpha matrix.Matrix, opts ...linalg.Opt) (err error) {

	params, e := linalg.GetParameters(opts...)
	if e != nil {
		err = e
		return
	}
	ind := linalg.GetIndexOpts(opts...)
	err = check_level3_func(ind, ftrmm, A, B, nil, params)
	if err != nil {
		return
	}
	if ! matrix.EqualTypes(A, B) {
		return errors.New("Parameters not of same type")
	}
	switch A.(type) {
	case *matrix.FloatMatrix:
		Aa := A.FloatArray()
		Ba := B.FloatArray()
		aval := 1.0
		if alpha != nil {
			aval = alpha.FloatValue()
		}
		if math.IsNaN(aval)  {
			return errors.New("alpha  not a number")
		}
		uplo := linalg.ParamString(params.Uplo)
		transA := linalg.ParamString(params.TransA)
		side := linalg.ParamString(params.Side)
		diag := linalg.ParamString(params.Diag)
		dtrmm(side, uplo, transA, diag,	ind.M, ind.N, aval,
			Aa[ind.OffsetA:], ind.LDa, Ba[ind.OffsetB:], ind.LDb)
	case *matrix.ComplexMatrix:
		Aa := A.ComplexArray()
		Ba := B.ComplexArray()
		aval := complex(1.0, 0.0)
		if alpha != nil {
			aval = alpha.ComplexValue()
			if cmplx.IsNaN(aval) {
				aval = complex(alpha.FloatValue(), 0.0)
			} else {
				return errors.New("alpha  not a number")
			}
		}
		uplo := linalg.ParamString(params.Uplo)
		transA := linalg.ParamString(params.TransA)
		side := linalg.ParamString(params.Side)
		diag := linalg.ParamString(params.Diag)
		ztrmm(side, uplo, transA, diag,	ind.M, ind.N, aval,
			Aa[ind.OffsetA:], ind.LDa, Ba[ind.OffsetB:], ind.LDb)
	default:
		return errors.New("Unknown type, not implemented")
	}
	return
}

/*
 Solution of a triangular system of equations with multiple righthand sides. (L3)

 Trsm(A, B, alpha=1.0, side=PLeft, uplo=PLower, transA=PNoTrans, diag=PNonUnit, 
 m=-1, n=-1, ldA=max(1,A.Rows), ldB=max(1,B.Rows), offsetA=0, offsetB=0)

 Computes
  B := alpha*A^{-1}*B if transA is PNoTrans   and side = PLeft
  B := alpha*B*A^{-1} if transA is PNoTrans   and side = PRight
  B := alpha*A^{-T}*B if transA is PTrans     and side = PLeft
  B := alpha*B*A^{-T} if transA is PTrans     and side = PRight
  B := alpha*A^{-H}*B if transA is PConjTrans and side = PLeft
  B := alpha*B*A^{-H} if transA is PConjTrans and side = PRight

 B is m by n and A is triangular.  The code does not verify whether A is nonsingular.

 ARGUMENTS
  A         float or complex matrix.
  B         float or complex matrix.  Must have the same type as A.
  alpha     number (float or complex singleton matrix).  Complex alpha is only
            allowed if A is complex.

 OPTIONS
  side      PLeft or PRight
  uplo      PLower or PUpper
  transA    PNoTrans or PTrans
  diag      PNonUnit or PUnit
  m         integer.  If negative, the default value is used.
            The default value is m = A.Rows or if side == PRight m = B.Rows
            If the default value is used and side is PLeft, m must be equal to A.Cols.
  n         integer.  If negative, the default value is used.
            The default value is n = B.Cols or if side )= PRight n = A.Rows.
            If the default value is used and side is PRight, n must be equal to A.Cols.
  ldA       nonnegative integer.
            ldA >= max(1,m) of if  side == PRight lda >= max(1,n).
            If zero, the default value is used. 
  ldB       nonnegative integer.  ldB >= max(1,m).
            If zero, the default value is used.
  offsetA   nonnegative integer
  offsetB   nonnegative integer
 */
func Trsm(A, B, alpha matrix.Matrix, opts ...linalg.Opt) (err error) {

	params, e := linalg.GetParameters(opts...)
	if e != nil {
		err = e
		return
	}
	ind := linalg.GetIndexOpts(opts...)
	err = check_level3_func(ind, ftrsm, A, B, nil, params)
	if err != nil {
		return
	}
	if ! matrix.EqualTypes(A, B) {
		return errors.New("Parameters not of same type")
	}
	switch A.(type) {
	case *matrix.FloatMatrix:
		Aa := A.FloatArray()
		Ba := B.FloatArray()
		aval := alpha.FloatValue()
		if math.IsNaN(aval)  {
			return errors.New("alpha or beta not a number")
		}
		uplo := linalg.ParamString(params.Uplo)
		transA := linalg.ParamString(params.TransA)
		side := linalg.ParamString(params.Side)
		diag := linalg.ParamString(params.Diag)
		dtrsm(side, uplo, transA, diag,	ind.M, ind.N, aval,
			Aa[ind.OffsetA:], ind.LDa, Ba[ind.OffsetB:], ind.LDb)
	case *matrix.ComplexMatrix:
		Aa := A.ComplexArray()
		Ba := B.ComplexArray()
		aval := complex(1.0, 0.0)
		if alpha != nil {
			aval = alpha.ComplexValue()
			if cmplx.IsNaN(aval) {
				aval = complex(alpha.FloatValue(), 0.0)
			} else {
				return errors.New("alpha  not a number")
			}
		}
		uplo := linalg.ParamString(params.Uplo)
		transA := linalg.ParamString(params.TransA)
		side := linalg.ParamString(params.Side)
		diag := linalg.ParamString(params.Diag)
		ztrsm(side, uplo, transA, diag,	ind.M, ind.N, aval,
			Aa[ind.OffsetA:], ind.LDa, Ba[ind.OffsetB:], ind.LDb)
	default:
		return errors.New("Unknown type, not implemented")
	}
	return
}


// Local Variables:
// tab-width: 4
// End: