// From: https://raw.githubusercontent.com/jhermsmeier/node-sift-distance/SIFT4/lib/sift.js
// SPDX-License-Identifier: MIT

/**
 * SIFT 4
 * @see http://siderite.blogspot.com/2014/11/super-fast-and-accurate-string-distance.html
 * @param  {String|Buffer|Array} s1
 * @param  {String|Buffer|Array} s2
 * @param  {Object} options
 *   @property {Number} maxOffset
 *   @property {Number} maxDistance
 *   @property {Function} tokenizer
 *   @property {Function} tokenMatcher
 *   @property {Function} matchEvaluator
 *   @property {Function} lengthEvaluator
 *   @property {Function} transpositionEvaluator
 * @return {Number}
 */
function SIFT( s1, s2, options ) {

  options = options != null ?
    options : {}

  var maxDistance = options.maxDistance
  var maxOffset   = options.maxOffset || SIFT.maxOffset
  var tokenize    = options.tokenizer || SIFT.tokenizer
  var match       = options.tokenMatcher || SIFT.tokenMatcher
  var evalMatch   = options.matchEvaluator || SIFT.matchEvaluator
  var evalLength  = options.lengthEvaluator || SIFT.lengthEvaluator
  var evalTrans   = options.transpositionEvaluator || SIFT.transpositionEvaluator

  var t1 = tokenize( s1 )
  var t2 = tokenize( s2 )

  var tl1 = t1.length
  var tl2 = t2.length

  if( tl1 === 0 ) return evalLength( tl2 )
  if( tl2 === 0 ) return evalLength( tl1 )

  // Cursors
  var c1 = 0
  var c2 = 0
  // Largest common subsequence
  var lcss = 0
  // Largest common substring
  var lcs = 0
  // Number of transpositions
  var trans = 0
  // Offset pair array
  var offsets = []

  while( ( c1 < tl1 ) && ( c2 < tl2 ) ) {
    if( match( t1[c1], t2[c2] ) ) {
      lcs = lcs + evalMatch( t1[c1], t2[c2] )
      while( offsets.length ) {
        if( c1 <= offsets[0][0] || c2 <= offsets[0][1] ) {
          trans++
          break
        } else {
          offsets.shift()
        }
      }
      offsets.push( [ c1, c2 ] )
    } else {
      lcss = lcss + evalLength( lcs )
      lcs = 0
      if( c1 !== c2 ) {
        c1 = c2 = Math.min( c1, c2 )
      }
      for( var i = 0; i < maxOffset; i++ ) {
        if( ( c1 + i < tl1 ) && match( t1[c1+i], t2[c2] ) ) {
          c1 = c1 + i - 1
          c2 = c2 - 1
          break
        }
        if( ( c2 + i < tl2 ) && match( t1[c1], t2[c2+i] ) ) {
          c1 = c1 - 1
          c2 = c2 + i - 1
          break
        }
      }
    }

    c1++
    c2++

    if( maxDistance ) {
      var distance = evalLength( Math.max( c1, c2 ) ) - evalTrans( lcss, trans )
      if( distance >= maxDistance ) return Math.round( distance )
    }

  }

  lcss = lcss + evalLength( lcs )

  return Math.round(
    evalLength( Math.max( tl1, tl2 ) ) -
    evalTrans( lcss, trans )
  )

}

/**
 * Default maximum lcs length
 * @type {Number}
 */
SIFT.maxOffset = 5

/**
 * Default tokenizer function
 * @param  {String|Buffer|Array} s
 * @return {String|Buffer|Array}
 */
SIFT.tokenizer = function( s ) {
  return s != null ? s : []
}

/**
 * Default token matcher
 * @param  {Mixed} t1
 * @param  {Mixed} t2
 * @return {Boolean}
 */
SIFT.tokenMatcher = function( t1, t2 ) {
  return t1 === t2
}

/**
 * Default match evaluator
 * @param  {Mixed} t1
 * @param  {Mixed} t2
 * @return {Number}
 */
SIFT.matchEvaluator = function( t1, t2 ) {
  return 1
}

/**
 * Default largest common substring length evaluator
 * @param  {Number} lcs
 * @return {Number}
 */
SIFT.lengthEvaluator = function( lcs ) {
  return lcs
}

/**
 * Default transposition count evalutator
 * @param  {Number} lcss
 * @param  {Number} trans
 * @return {Number}
 */
SIFT.transpositionEvaluator = function( lcss, trans ) {
  return lcss - trans / 2
}

// Exports
//module.exports = SIFT
export {SIFT};
