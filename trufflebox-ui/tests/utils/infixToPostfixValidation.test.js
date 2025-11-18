/**
 * Comprehensive Infix-to-Postfix Expression Validation Test Suite
 * 
 * This test suite validates the JavaScript implementation of infix-to-postfix
 * expression conversion against a comprehensive set of test cases equivalent
 * to the Go implementation test suite.
 * 
 * Features tested:
 * - Function recognition and arity validation
 * - Operator precedence and associativity
 * - Expression parsing and validation
 * - Infix to postfix conversion
 * - Complex edge cases and nested expressions
 * 
 * Usage:
 *   node tests/utils/infixToPostfixValidation.test.js
 * 
 * or from project root:
 *   npm test -- tests/utils/infixToPostfixValidation.test.js
 * 
 * @version 1.0.0
 * @author TruffleBox Team
 * @license MIT
 */

import { 
  validateInfixExpression, 
  convertInfixToPostfix, 
  validatePostfixExpression,
  getAvailableFunctions,
  getAvailableOperators,
  getFunctionArity 
} from '../../src/utils/infixToPostfix.js';

// Test configuration
const TEST_CONFIG = {
  displayVerbose: process.env.TEST_VERBOSE === 'true',
  exitOnFailure: process.env.TEST_EXIT_ON_FAILURE === 'true'
};

// Helper functions
const arraysEqual = (a, b) => {
  if (a.length !== b.length) return false;
  return a.every((val, index) => val === b[index]);
};

const logVerbose = (message) => {
  if (TEST_CONFIG.displayVerbose) {
    console.log(message);
  }
};

const logTestResult = (passed, testName, details = '') => {
  const icon = passed ? '✅' : '❌';
  const message = `${icon} ${testName}${details ? ': ' + details : ''}`;
  
  if (passed) {
    logVerbose(message);
  } else {
    console.log(message);
  }
};

// Test suite runner
class TestSuite {
  constructor(name) {
    this.name = name;
    this.passed = 0;
    this.total = 0;
    this.failures = [];
  }

  test(name, testFn) {
    this.total++;
    try {
      const result = testFn();
      if (result.passed) {
        this.passed++;
        logTestResult(true, name, result.details);
      } else {
        this.failures.push({ name, ...result });
        logTestResult(false, name, result.details);
      }
    } catch (error) {
      this.failures.push({ name, error: error.message });
      logTestResult(false, name, `Error: ${error.message}`);
    }
  }

  getResults() {
    return {
      name: this.name,
      passed: this.passed,
      total: this.total,
      failures: this.failures,
      percentage: ((this.passed / this.total) * 100).toFixed(1)
    };
  }
}

// Main test execution
console.log('🧪 COMPREHENSIVE INFIX-POSTFIX VALIDATION TEST SUITE');
console.log('=' .repeat(80));
console.log('Testing JavaScript implementation against Go test case equivalents');
console.log('');

// Test 1: Function Recognition
console.log('1️⃣  FUNCTION RECOGNITION TESTS');
const functionSuite = new TestSuite('Function Recognition');

const functionTests = [
  // Valid functions
  { token: 'exp', expected: true, name: 'Exponential function' },
  { token: 'log', expected: true, name: 'Logarithm function' },
  { token: 'abs', expected: true, name: 'Absolute value function' },
  { token: 'norm_min_max', expected: true, name: 'Normalize min-max function' },
  { token: 'percentile_rank', expected: true, name: 'Percentile rank function' },
  { token: 'norm_percentile_0_99', expected: true, name: 'Normalize percentile 0-99 function' },
  { token: 'norm_percentile_5_95', expected: true, name: 'Normalize percentile 5-95 function' },
  { token: 'min', expected: true, name: 'Minimum function' },
  { token: 'max', expected: true, name: 'Maximum function' },
  
  // Invalid functions  
  { token: '+', expected: false, name: 'Addition operator (not function)' },
  { token: 'variable', expected: false, name: 'Variable name (not function)' },
  { token: '123', expected: false, name: 'Number (not function)' },
  { token: '', expected: false, name: 'Empty string (not function)' },
  { token: 'EXP', expected: false, name: 'Case sensitive check (uppercase)' },
  { token: 'exp2', expected: false, name: 'Invalid function name with suffix' }
];

const availableFunctions = getAvailableFunctions();
functionTests.forEach(test => {
  functionSuite.test(test.name, () => {
    const result = availableFunctions.includes(test.token);
    return {
      passed: result === test.expected,
      details: `"${test.token}" -> ${result} (expected ${test.expected})`
    };
  });
});

// Test 2: Operator Recognition
console.log('\n2️⃣  OPERATOR RECOGNITION TESTS');
const operatorSuite = new TestSuite('Operator Recognition');

const operatorTests = [
  // Valid operators
  { token: '+', expected: true, name: 'Addition operator' },
  { token: '-', expected: true, name: 'Subtraction operator' },
  { token: '*', expected: true, name: 'Multiplication operator' },
  { token: '/', expected: true, name: 'Division operator' },
  { token: '^', expected: true, name: 'Power operator' },
  { token: '>', expected: true, name: 'Greater than operator' },
  { token: '<', expected: true, name: 'Less than operator' },
  { token: '>=', expected: true, name: 'Greater than or equal operator' },
  { token: '<=', expected: true, name: 'Less than or equal operator' },
  { token: '==', expected: true, name: 'Equality operator' },
  { token: '&', expected: true, name: 'Logical AND operator' },
  { token: '|', expected: true, name: 'Logical OR operator' },
  
  // Invalid operators
  { token: 'variable', expected: false, name: 'Variable name (not operator)' },
  { token: '123', expected: false, name: 'Number (not operator)' },
  { token: '(', expected: false, name: 'Open parenthesis (not operator)' },
  { token: ')', expected: false, name: 'Close parenthesis (not operator)' },
  { token: ',', expected: false, name: 'Comma (not operator)' },
  { token: '', expected: false, name: 'Empty string (not operator)' },
  { token: '++', expected: false, name: 'Invalid double operator' }
];

const availableOperators = getAvailableOperators();
operatorTests.forEach(test => {
  operatorSuite.test(test.name, () => {
    const result = availableOperators.includes(test.token);
    return {
      passed: result === test.expected,
      details: `"${test.token}" -> ${result} (expected ${test.expected})`
    };
  });
});

// Test 3: Function Arity Validation
console.log('\n3️⃣  FUNCTION ARITY VALIDATION TESTS');
const aritySuite = new TestSuite('Function Arity');

const arityTests = [
  // Unary functions (arity 1)
  { func: 'exp', expected: 1, name: 'Exponential function arity' },
  { func: 'log', expected: 1, name: 'Logarithm function arity' },
  { func: 'abs', expected: 1, name: 'Absolute value function arity' },
  { func: 'norm_min_max', expected: 1, name: 'Normalize min-max function arity' },
  { func: 'percentile_rank', expected: 1, name: 'Percentile rank function arity' },
  { func: 'norm_percentile_0_99', expected: 1, name: 'Normalize percentile 0-99 function arity' },
  { func: 'norm_percentile_5_95', expected: 1, name: 'Normalize percentile 5-95 function arity' },
  
  // Binary functions (arity 2)
  { func: 'min', expected: 2, name: 'Minimum function arity' },
  { func: 'max', expected: 2, name: 'Maximum function arity' },
  
  // Invalid functions
  { func: 'invalid', expected: 0, name: 'Invalid function arity' }
];

arityTests.forEach(test => {
  aritySuite.test(test.name, () => {
    const result = getFunctionArity(test.func);
    return {
      passed: result === test.expected,
      details: `"${test.func}" -> arity ${result} (expected ${test.expected})`
    };
  });
});

// Test 4: Expression Validation
console.log('\n4️⃣  EXPRESSION VALIDATION TESTS');
const validationSuite = new TestSuite('Expression Validation');

const validationTests = [
  // Valid expressions
  { expr: 'x + y', expected: true, name: 'Simple addition expression' },
  { expr: 'x - y', expected: true, name: 'Simple subtraction expression' },
  { expr: 'x * y', expected: true, name: 'Simple multiplication expression' },
  { expr: 'x / y', expected: true, name: 'Simple division expression' },
  { expr: 'x ^ y', expected: true, name: 'Simple power expression' },
  { expr: '(x + y) * z', expected: true, name: 'Expression with parentheses' },
  { expr: 'exp(x)', expected: true, name: 'Unary function call' },
  { expr: 'min(x, y)', expected: true, name: 'Binary function call' },
  { expr: 'x + y * z - w / u', expected: true, name: 'Complex arithmetic expression' },
  { expr: 'exp(log(x))', expected: true, name: 'Nested function calls' },
  { expr: 'x > y', expected: true, name: 'Comparison expression' },
  { expr: 'x & y', expected: true, name: 'Logical AND expression' },
  { expr: 'x > y & z < w', expected: true, name: 'Complex logical expression' },
  { expr: 'x + 123', expected: true, name: 'Expression with numbers' },
  { expr: '123 + 456', expected: true, name: 'Numeric expression' },
  { expr: 'x + -123', expected: true, name: 'Expression with negative numbers' },
  { expr: 'x', expected: true, name: 'Single variable' },
  { expr: '123', expected: true, name: 'Single number' },
  
  // Invalid expressions
  { expr: '', expected: false, name: 'Empty expression' },
  { expr: 'x + (y', expected: false, name: 'Mismatched parentheses (missing close)' },
  { expr: 'x + y)', expected: false, name: 'Mismatched parentheses (extra close)' },
  { expr: 'exp(x, y)', expected: false, name: 'Wrong function arity (too many args)' },
  { expr: 'min(x)', expected: false, name: 'Wrong function arity (too few args)' }
];

validationTests.forEach(test => {
  validationSuite.test(test.name, () => {
    const result = validateInfixExpression(test.expr);
    const passed = result.isValid === test.expected;
    let details = `"${test.expr}" -> ${result.isValid ? 'valid' : 'invalid'} (expected ${test.expected ? 'valid' : 'invalid'})`;
    
    if (!passed && !result.isValid) {
      details += ` | Errors: ${result.errors.join(', ')}`;
    }
    
    return { passed, details };
  });
});

// Test 5: Infix to Postfix Conversion
console.log('\n5️⃣  INFIX TO POSTFIX CONVERSION TESTS');
const conversionSuite = new TestSuite('Infix to Postfix Conversion');

const conversionTests = [
  // Basic expressions
  { infix: 'x + y', expected: ['x', 'y', '+'], name: 'Simple addition' },
  { infix: 'x - y', expected: ['x', 'y', '-'], name: 'Simple subtraction' },
  { infix: 'x * y', expected: ['x', 'y', '*'], name: 'Simple multiplication' },
  { infix: 'x / y', expected: ['x', 'y', '/'], name: 'Simple division' },
  { infix: 'x ^ y', expected: ['x', 'y', '^'], name: 'Simple power' },
  
  // Precedence tests
  { infix: 'x + y * z', expected: ['x', 'y', 'z', '*', '+'], name: 'Multiplication precedence over addition' },
  { infix: 'x - y / z', expected: ['x', 'y', 'z', '/', '-'], name: 'Division precedence over subtraction' },
  { infix: 'x * y ^ z', expected: ['x', 'y', 'z', '^', '*'], name: 'Power precedence over multiplication' },
  
  // Associativity tests
  { infix: 'x + y + z', expected: ['x', 'y', '+', 'z', '+'], name: 'Left associative addition' },
  { infix: 'x ^ y ^ z', expected: ['x', 'y', 'z', '^', '^'], name: 'Right associative power' },
  
  // Parentheses tests
  { infix: '(x + y)', expected: ['x', 'y', '+'], name: 'Simple parentheses' },
  { infix: '(x + y) * z', expected: ['x', 'y', '+', 'z', '*'], name: 'Parentheses changing precedence' },
  { infix: '(x + y) * (z - w)', expected: ['x', 'y', '+', 'z', 'w', '-', '*'], name: 'Multiple parentheses groups' },
  
  // Function tests
  { infix: 'exp(x)', expected: ['x', 'exp'], name: 'Unary function conversion' },
  { infix: 'min(x, y)', expected: ['x', 'y', 'min'], name: 'Binary function conversion' },
  { infix: 'exp(log(x))', expected: ['x', 'log', 'exp'], name: 'Nested function conversion' },
  { infix: 'x + exp(y)', expected: ['x', 'y', 'exp', '+'], name: 'Function in arithmetic expression' },
  
  // Comparison operators
  { infix: 'x > y', expected: ['x', 'y', '>'], name: 'Greater than comparison' },
  { infix: 'x >= y', expected: ['x', 'y', '>='], name: 'Greater than or equal comparison' },
  { infix: 'x + y > z', expected: ['x', 'y', '+', 'z', '>'], name: 'Comparison with arithmetic' },
  
  // Logical operators
  { infix: 'x & y', expected: ['x', 'y', '&'], name: 'Logical AND' },
  { infix: 'x | y', expected: ['x', 'y', '|'], name: 'Logical OR' },
  { infix: 'x + y > z & w < u', expected: ['x', 'y', '+', 'z', '>', 'w', 'u', '<', '&'], name: 'Complex logical with comparisons' },
  
  // Complex expressions
  { infix: 'exp(log(x) + y) * z', expected: ['x', 'log', 'y', '+', 'exp', 'z', '*'], name: 'Complex mathematical expression' },
  { infix: 'min(max(x, y), z)', expected: ['x', 'y', 'max', 'z', 'min'], name: 'Nested min/max functions' }
];

conversionTests.forEach(test => {
  conversionSuite.test(test.name, () => {
    const result = convertInfixToPostfix(test.infix);
    if (result.success) {
      const postfixArray = result.postfix.split(' ').filter(token => token.length > 0);
      const passed = arraysEqual(postfixArray, test.expected);
      const details = `"${test.infix}" -> [${postfixArray.join(', ')}] (expected [${test.expected.join(', ')}])`;
      return { passed, details };
    } else {
      return {
        passed: false,
        details: `"${test.infix}" -> CONVERSION FAILED | Errors: ${result.errors.join(', ')}`
      };
    }
  });
});

// Test 6: Complex Edge Cases
console.log('\n6️⃣  COMPLEX EDGE CASE TESTS');
const edgeCaseSuite = new TestSuite('Complex Edge Cases');

const edgeCaseTests = [
  // Precedence edge cases
  { 
    infix: 'x | y & z == w > u + v * w ^ z', 
    name: 'All precedence levels interaction',
    type: 'conversion'
  },
  { 
    infix: 'x ^ y ^ z ^ w', 
    expected: ['x', 'y', 'z', 'w', '^', '^', '^'], 
    name: 'Right associative power chain',
    type: 'conversion'
  },
  { 
    infix: 'x - y - z - w', 
    expected: ['x', 'y', '-', 'z', '-', 'w', '-'], 
    name: 'Left associative subtraction chain',
    type: 'conversion'
  },
  
  // Complex nesting
  { 
    infix: '((((x + y) * z) - w) / u)', 
    expected: ['x', 'y', '+', 'z', '*', 'w', '-', 'u', '/'], 
    name: 'Deeply nested parentheses',
    type: 'conversion'
  },
  { 
    infix: 'exp(log(abs(min(x, y))))', 
    expected: ['x', 'y', 'min', 'abs', 'log', 'exp'], 
    name: 'Complex function nesting',
    type: 'conversion'
  },
  
  // Function arity validation
  { 
    infix: 'exp(x, y)', 
    expected: false, 
    name: 'Exponential function with wrong arity',
    type: 'validation'
  },
  { 
    infix: 'min(x)', 
    expected: false, 
    name: 'Minimum function with insufficient arguments',
    type: 'validation'
  },
  { 
    infix: 'max(x, y, z)', 
    expected: false, 
    name: 'Maximum function with too many arguments',
    type: 'validation'
  }
];

edgeCaseTests.forEach(test => {
  edgeCaseSuite.test(test.name, () => {
    if (test.type === 'validation') {
      const result = validateInfixExpression(test.infix);
      const passed = result.isValid === test.expected;
      const details = `"${test.infix}" -> ${result.isValid ? 'valid' : 'invalid'} (expected ${test.expected ? 'valid' : 'invalid'})`;
      return { passed, details };
    } else {
      const result = convertInfixToPostfix(test.infix);
      if (result.success) {
        if (test.expected) {
          const postfixArray = result.postfix.split(' ').filter(token => token.length > 0);
          const passed = arraysEqual(postfixArray, test.expected);
          const details = `"${test.infix}" -> [${postfixArray.join(', ')}] (expected [${test.expected.join(', ')}])`;
          return { passed, details };
        } else {
          const details = `"${test.infix}" -> [${result.postfix.split(' ').join(', ')}] (SUCCESS)`;
          return { passed: true, details };
        }
      } else {
        return {
          passed: false,
          details: `"${test.infix}" -> CONVERSION FAILED | Errors: ${result.errors.join(', ')}`
        };
      }
    }
  });
});

// Collect all results
const allSuites = [
  functionSuite,
  operatorSuite,
  aritySuite,
  validationSuite,
  conversionSuite,
  edgeCaseSuite
];

const allResults = allSuites.map(suite => suite.getResults());

// Final Summary
console.log('\n🎯 FINAL TEST RESULTS SUMMARY');
console.log('=' .repeat(80));

let totalPassed = 0;
let totalTests = 0;
let hasFailures = false;

allResults.forEach(result => {
  totalPassed += result.passed;
  totalTests += result.total;
  
  console.log(`📊 ${result.name}: ${result.passed}/${result.total} tests passed (${result.percentage}%)`);
  
  if (result.failures.length > 0) {
    hasFailures = true;
    if (!TEST_CONFIG.displayVerbose) {
      console.log('   Failed tests:');
      result.failures.forEach(failure => {
        console.log(`   ❌ ${failure.name}: ${failure.details || failure.error}`);
      });
    }
  }
});

console.log('\n' + '=' .repeat(80));
console.log(`✨ OVERALL RESULTS: ${totalPassed}/${totalTests} tests passed (${((totalPassed/totalTests)*100).toFixed(1)}%)`);

if (totalPassed === totalTests) {
  console.log('\n🎉 ALL TESTS PASSED! The implementation is working correctly! 🎉');
  console.log('✅ Your infix-to-postfix converter is ready for production use.');
} else {
  console.log(`\n⚠️  ${totalTests - totalPassed} tests failed. Please review the implementation.`);
  console.log('❌ Some functionality may need to be fixed before production use.');
  
  if (TEST_CONFIG.exitOnFailure) {
    process.exit(1);
  }
}

console.log('\n📝 Test completed. Thank you for using the TruffleBox validation suite!'); 