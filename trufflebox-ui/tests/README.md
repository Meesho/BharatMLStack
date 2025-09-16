# TruffleBox Test Suite

This directory contains comprehensive test suites for validating the TruffleBox implementation. The tests are designed to ensure correctness, reliability, and compatibility with the original Go implementation.

## Test Structure

```
tests/
├── README.md                           # This file
└── utils/
    └── infixToPostfixValidation.test.js # Comprehensive infix-to-postfix validation
```

## Available Tests

### 1. Infix-to-Postfix Validation Test Suite

**File:** `utils/infixToPostfixValidation.test.js`

This comprehensive test suite validates the JavaScript implementation of infix-to-postfix expression conversion against test cases equivalent to the Go implementation.

**Features Tested:**
- ✅ Function recognition and arity validation
- ✅ Operator precedence and associativity
- ✅ Expression parsing and validation
- ✅ Infix to postfix conversion
- ✅ Complex edge cases and nested expressions

**Test Categories:**
1. **Function Recognition** (15 tests) - Validates proper recognition of mathematical functions
2. **Operator Recognition** (19 tests) - Validates binary and unary operator detection
3. **Function Arity** (10 tests) - Validates function argument count validation
4. **Expression Validation** (23 tests) - Validates complex expression parsing
5. **Infix to Postfix Conversion** (25 tests) - Validates conversion accuracy
6. **Complex Edge Cases** (8 tests) - Validates advanced scenarios

**Total:** 100 comprehensive test cases

## Running Tests

### Quick Start

```bash
# From project root
node tests/utils/infixToPostfixValidation.test.js
```

### With Verbose Output

```bash
# Show all test results (including passed tests)
TEST_VERBOSE=true node tests/utils/infixToPostfixValidation.test.js
```

### With Exit on Failure

```bash
# Exit with code 1 if any tests fail (useful for CI/CD)
TEST_EXIT_ON_FAILURE=true node tests/utils/infixToPostfixValidation.test.js
```

### Combined Options

```bash
# Verbose output with exit on failure
TEST_VERBOSE=true TEST_EXIT_ON_FAILURE=true node tests/utils/infixToPostfixValidation.test.js
```

## Test Output

### Successful Run
```
🧪 COMPREHENSIVE INFIX-POSTFIX VALIDATION TEST SUITE
================================================================================
Testing JavaScript implementation against Go test case equivalents

1️⃣  FUNCTION RECOGNITION TESTS
2️⃣  OPERATOR RECOGNITION TESTS
3️⃣  FUNCTION ARITY VALIDATION TESTS
4️⃣  EXPRESSION VALIDATION TESTS
5️⃣  INFIX TO POSTFIX CONVERSION TESTS
6️⃣  COMPLEX EDGE CASE TESTS

🎯 FINAL TEST RESULTS SUMMARY
================================================================================
📊 Function Recognition: 15/15 tests passed (100.0%)
📊 Operator Recognition: 19/19 tests passed (100.0%)
📊 Function Arity: 10/10 tests passed (100.0%)
📊 Expression Validation: 23/23 tests passed (100.0%)
📊 Infix to Postfix Conversion: 25/25 tests passed (100.0%)
📊 Complex Edge Cases: 8/8 tests passed (100.0%)

================================================================================
✨ OVERALL RESULTS: 100/100 tests passed (100.0%)

🎉 ALL TESTS PASSED! The implementation is working correctly! 🎉
✅ Your infix-to-postfix converter is ready for production use.
```

### Failed Tests
When tests fail, detailed error information is provided:
```
❌ Wrong function arity (too many args): "exp(x, y)" -> invalid (expected invalid) | Errors: Function exp expects 1 argument(s), got 2
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `TEST_VERBOSE` | Show all test results including passed tests | `false` |
| `TEST_EXIT_ON_FAILURE` | Exit with code 1 if any tests fail | `false` |

## Supported Functions

The test suite validates the following mathematical functions:

### Unary Functions (1 argument)
- `exp(x)` - Exponential function
- `log(x)` - Natural logarithm
- `abs(x)` - Absolute value
- `norm_min_max(x)` - Min-max normalization
- `percentile_rank(x)` - Percentile rank
- `norm_percentile_0_99(x)` - Normalize to 0-99 percentile range
- `norm_percentile_5_95(x)` - Normalize to 5-95 percentile range

### Binary Functions (2 arguments)
- `min(x, y)` - Minimum value
- `max(x, y)` - Maximum value

## Supported Operators

The test suite validates these operators with proper precedence and associativity:

### Arithmetic Operators
- `+` - Addition (left associative, precedence 3)
- `-` - Subtraction (left associative, precedence 3)
- `*` - Multiplication (left associative, precedence 4)
- `/` - Division (left associative, precedence 4)
- `^` - Exponentiation (right associative, precedence 5)

### Comparison Operators
- `>` - Greater than (left associative, precedence 2)
- `<` - Less than (left associative, precedence 2)
- `>=` - Greater than or equal (left associative, precedence 2)
- `<=` - Less than or equal (left associative, precedence 2)
- `==` - Equal (left associative, precedence 2)

### Logical Operators
- `&` - Logical AND (left associative, precedence 1)
- `|` - Logical OR (left associative, precedence 0)

## Test Examples

### Basic Expressions
```javascript
'x + y' → ['x', 'y', '+']
'x * y ^ z' → ['x', 'y', 'z', '^', '*']
'(x + y) * z' → ['x', 'y', '+', 'z', '*']
```

### Function Calls
```javascript
'exp(x)' → ['x', 'exp']
'min(x, y)' → ['x', 'y', 'min']
'exp(log(x))' → ['x', 'log', 'exp']
```

### Complex Expressions
```javascript
'exp(log(x) + y) * z' → ['x', 'log', 'y', '+', 'exp', 'z', '*']
'x + y > z & w < u' → ['x', 'y', '+', 'z', '>', 'w', 'u', '<', '&']
```

## Contributing

When adding new functionality to the infix-to-postfix converter:

1. **Run existing tests** to ensure no regressions
2. **Add new test cases** for new features
3. **Update documentation** if new operators or functions are added
4. **Ensure all tests pass** before submitting PRs

### Adding New Tests

To add new test cases, modify the appropriate test array in `infixToPostfixValidation.test.js`:

```javascript
// Add to functionTests array for new functions
{ token: 'newfunction', expected: true, name: 'New function name' }

// Add to operatorTests array for new operators
{ token: '@@', expected: true, name: 'New operator name' }

// Add to conversionTests array for new conversion examples
{ infix: 'new expression', expected: ['expected', 'postfix'], name: 'Test description' }
```

## License

This test suite is part of the TruffleBox project and is released under the MIT License.

## Support

For questions or issues with the test suite, please open an issue in the TruffleBox repository.

---

**Made with ❤️ by the TruffleBox(Powered by Meesho) Team** 