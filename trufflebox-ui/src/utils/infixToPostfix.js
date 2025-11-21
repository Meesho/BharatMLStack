// Operator precedence
const binaryOpsPrecedence = {
  "^": 5,
  "*": 4,
  "/": 4,
  "+": 3,
  "-": 3,
  ">": 2,
  "<": 2,
  ">=": 2,
  "<=": 2,
  "==": 2,
  "&": 1,
  "|": 1,
};

// Function arity
const funcArity = {
  "exp": 1,
  "log": 1,
  "abs": 1,
  "min": 2,
  "max": 2,
  "norm_min_max": 1,
  "percentile_rank": 1,
  "norm_percentile_0_99": 1,
  "norm_percentile_5_95": 1,
};

const isOperator = (token) => {
  return binaryOpsPrecedence.hasOwnProperty(token);
};

const isFunction = (token) => {
  return funcArity.hasOwnProperty(token);
};

const isNumber = (token) => {
  // Support integers and decimals, but not scientific notation
  return /^-?\d+(\.\d+)?$/.test(token);
};

const isVariable = (token) => {
  // Variables should be alphanumeric and underscore, not matching function names
  return /^[a-zA-Z_][a-zA-Z0-9_]*$/.test(token) && !isFunction(token);
};

const tokenize = (expression) => {
  const tokens = [];
  let i = 0;
  
  while (i < expression.length) {
    const char = expression[i];
    
    // Skip whitespace
    if (/\s/.test(char)) {
      i++;
      continue;
    }
    
    // Handle multi-character operators
    if (i < expression.length - 1) {
      const twoChar = expression.substr(i, 2);
      if ([">=", "<=", "=="].includes(twoChar)) {
        tokens.push(twoChar);
        i += 2;
        continue;
      }
    }
    
    // Handle parentheses and most operators first (but not minus, which needs special handling)
    if (["(", ")", "+", "*", "/", "^", ">", "<", "&", "|", ","].includes(char)) {
      tokens.push(char);
      i++;
      continue;
    }
    
    // Handle minus separately 
    if (char === '-') {
      // Check if this is a negative number (minus after (, operator, or at start)
      const prevChar = i > 0 ? expression[i - 1] : null;
      const nextChar = i + 1 < expression.length ? expression[i + 1] : null;
      
      // Look at the actual previous character, not just the previous token
      let prevNonSpace = null;
      let j = i - 1;
      while (j >= 0 && /\s/.test(expression[j])) {
        j--;
      }
      if (j >= 0) {
        prevNonSpace = expression[j];
      }
      
      if ((prevNonSpace === null || prevNonSpace === '(' || [">=", "<=", "==", "+", "-", "*", "/", "^", ">", "<", "&", "|", ","].includes(prevNonSpace)) &&
          nextChar && /\d/.test(nextChar)) {
        // This looks like a negative number
        let numStr = '-';
        i++;
        while (i < expression.length && /[\d.]/.test(expression[i])) {
          numStr += expression[i];
          i++;
        }
        tokens.push(numStr);
        continue;
      } else {
        // This is a binary minus operator
        tokens.push(char);
        i++;
        continue;
      }
    }
    
    // Handle positive numbers
    if (/\d/.test(char)) {
      let numStr = '';
      while (i < expression.length && /[\d.]/.test(expression[i])) {
        numStr += expression[i];
        i++;
      }
      tokens.push(numStr);
      continue;
    }
    
    // Handle variables and functions
    if (/[a-zA-Z_]/.test(char)) {
      let identifier = '';
      while (i < expression.length && /[a-zA-Z0-9_]/.test(expression[i])) {
        identifier += expression[i];
        i++;
      }
      tokens.push(identifier);
      continue;
    }
    
    throw new Error(`Invalid character: ${char}`);
  }
  
  return tokens;
};

export const validateInfixExpression = (expression) => {
  const errors = [];
  
  try {
    const tokens = tokenize(expression);
    
    // Check for empty expression
    if (tokens.length === 0) {
      errors.push("Expression cannot be empty");
      return { isValid: false, errors };
    }
    
    // Check for invalid tokens
    for (let i = 0; i < tokens.length; i++) {
      const token = tokens[i];
      
      if (!isOperator(token) && !isFunction(token) && !isNumber(token) && 
          !isVariable(token) && !["(", ")", ","].includes(token)) {
        errors.push(`Invalid token: ${token}`);
      }
    }
    

    
    // Check for function usage
    for (let i = 0; i < tokens.length; i++) {
      const token = tokens[i];
      if (isFunction(token)) {
        // Check if followed by opening parenthesis
        if (i + 1 >= tokens.length || tokens[i + 1] !== '(') {
          errors.push(`Function ${token} must be followed by parentheses`);
        }
      }
    }
    
    // Check for balanced parentheses
    let parenCount = 0;
    for (const token of tokens) {
      if (token === '(') parenCount++;
      if (token === ')') parenCount--;
      if (parenCount < 0) {
        errors.push("Unmatched closing parenthesis");
        break;
      }
    }
    if (parenCount > 0) {
      errors.push("Unmatched opening parenthesis");
    }
    
    const semanticErrors = validateSemantics(tokens);
    errors.push(...semanticErrors);
    
    return { isValid: errors.length === 0, errors };
  } catch (error) {
    return { isValid: false, errors: [error.message] };
  }
};

const validateSemantics = (tokens) => {
  const errors = [];
  
  // Check for operators without sufficient operands
  for (let i = 0; i < tokens.length; i++) {
    const token = tokens[i];
    
    if (isOperator(token)) {
      const prevToken = i > 0 ? tokens[i - 1] : null;
      const nextToken = i < tokens.length - 1 ? tokens[i + 1] : null;
      
      // Handle unary minus (- at start or after (, operator, or ,)
      if (token === '-' && (prevToken === null || prevToken === '(' || isOperator(prevToken) || prevToken === ',')) {
        // This is likely a unary minus, just check if it has a right operand
        if (nextToken === null || nextToken === ')' || nextToken === ',' || isOperator(nextToken)) {
          errors.push(`Not enough operands for unary operator '${token}'`);
        }
        continue;
      }
      
      // Handle binary operators
      // Check if operator has left operand
      if (prevToken === null || prevToken === '(' || isOperator(prevToken) || prevToken === ',') {
        errors.push(`Not enough operands for binary operator '${token}'`);
        continue;
      }
      
      // Check if operator has right operand
      if (nextToken === null || nextToken === ')' || isOperator(nextToken) || nextToken === ',') {
        errors.push(`Not enough operands for binary operator '${token}'`);
        continue;
      }
    }
  }
  
  // Check for consecutive operators
  for (let i = 0; i < tokens.length - 1; i++) {
    const token = tokens[i];
    const nextToken = tokens[i + 1];
    
    if (isOperator(token) && isOperator(nextToken)) {
      errors.push(`Consecutive operators '${token}' and '${nextToken}' are not allowed`);
    }
  }
  
  // Check for empty parentheses
  for (let i = 0; i < tokens.length - 1; i++) {
    if (tokens[i] === '(' && tokens[i + 1] === ')') {
      errors.push("Empty parentheses are not allowed");
    }
  }
  
  // Check for incomplete expressions (ending with operator)
  const lastToken = tokens[tokens.length - 1];
  if (isOperator(lastToken) || lastToken === ',') {
    errors.push("Expression cannot end with an operator or comma");
  }
  
  // Check for expressions starting with binary operators
  const firstToken = tokens[0];
  if (isOperator(firstToken)) {
    errors.push("Expression cannot start with a binary operator");
  }
  
  // Validate function calls
  errors.push(...validateFunctionCalls(tokens));
  
  // Check for missing operands around commas
  for (let i = 0; i < tokens.length; i++) {
    const token = tokens[i];
    if (token === ',') {
      const prevToken = i > 0 ? tokens[i - 1] : null;
      const nextToken = i < tokens.length - 1 ? tokens[i + 1] : null;
      
      if (prevToken === null || prevToken === '(' || prevToken === ',' || isOperator(prevToken)) {
        errors.push("Missing operand before comma");
      }
      if (nextToken === null || nextToken === ')' || nextToken === ',' || isOperator(nextToken)) {
        errors.push("Missing operand after comma");
      }
    }
  }
  
  return errors;
};

const validateFunctionCalls = (tokens) => {
  const errors = [];
  
  // Find all function calls and validate them individually
  for (let i = 0; i < tokens.length; i++) {
    const token = tokens[i];
    
    if (isFunction(token)) {
      // Expect opening parenthesis next
      if (i + 1 >= tokens.length || tokens[i + 1] !== '(') {
        errors.push(`Function ${token} must be followed by opening parenthesis`);
        continue;
      }
      
      // Find the matching closing parenthesis and parse arguments
      const funcName = token;
      const expectedArgs = getFunctionArity(funcName);
      let parenLevel = 0;
      let argCount = 0;
      let foundArg = false;
      let j = i + 1; // Start after function name
      
      while (j < tokens.length) {
        const currentToken = tokens[j];
        
        if (currentToken === '(') {
          parenLevel++;
          // If we're at the top level and haven't found an argument yet, this starts an argument
          if (parenLevel === 1 && !foundArg) {
            foundArg = true;
          }
        } else if (currentToken === ')') {
          parenLevel--;
          if (parenLevel === 0) {
            // Found the end of this function call
            if (foundArg) {
              argCount++; // Count the last argument
            }
            break;
          }
        } else if (currentToken === ',' && parenLevel === 1) {
          // This comma is at the top level of this function call
          if (!foundArg) {
            errors.push(`Missing argument before comma in function ${funcName}`);
          } else {
            argCount++;
            foundArg = false;
          }
        } else if (parenLevel === 1 && !foundArg) {
          // Any token at the top level starts an argument
          foundArg = true;
        }
        
        j++;
      }
      
      // Validate argument count
      if (argCount !== expectedArgs) {
        errors.push(`Function ${funcName} expects ${expectedArgs} argument(s), got ${argCount}`);
      }
      
      // If we expected arguments but found none
      if (expectedArgs > 0 && !foundArg && argCount === 0) {
        errors.push(`Function ${funcName} has no arguments but expects ${expectedArgs}`);
      }
    }
  }
  
  return errors;
};

export const convertInfixToPostfix = (expression) => {
  const validation = validateInfixExpression(expression);
  if (!validation.isValid) {
    return { success: false, errors: validation.errors };
  }
  
  try {
    const tokens = tokenize(expression);
    const output = [];
    const operatorStack = [];
    
    for (let i = 0; i < tokens.length; i++) {
      const token = tokens[i];
      
      if (isNumber(token) || isVariable(token)) {
        output.push(token);
      } else if (isFunction(token)) {
        operatorStack.push(token);
      } else if (token === ',') {
        // Pop operators until we find opening parenthesis
        while (operatorStack.length > 0 && operatorStack[operatorStack.length - 1] !== '(') {
          output.push(operatorStack.pop());
        }
      } else if (isOperator(token)) {
        while (operatorStack.length > 0 && 
               operatorStack[operatorStack.length - 1] !== '(' &&
               !isFunction(operatorStack[operatorStack.length - 1]) &&
               ((binaryOpsPrecedence[operatorStack[operatorStack.length - 1]] > binaryOpsPrecedence[token]) ||
                (binaryOpsPrecedence[operatorStack[operatorStack.length - 1]] === binaryOpsPrecedence[token] && token !== '^'))) {
          output.push(operatorStack.pop());
        }
        operatorStack.push(token);
      } else if (token === '(') {
        operatorStack.push(token);
      } else if (token === ')') {
        while (operatorStack.length > 0 && operatorStack[operatorStack.length - 1] !== '(') {
          output.push(operatorStack.pop());
        }
        if (operatorStack.length === 0) {
          throw new Error("Unmatched closing parenthesis");
        }
        operatorStack.pop(); // Remove the '('
        
        // If there's a function on top of stack, pop it
        if (operatorStack.length > 0 && isFunction(operatorStack[operatorStack.length - 1])) {
          output.push(operatorStack.pop());
        }
      }
    }
    
    // Pop remaining operators
    while (operatorStack.length > 0) {
      if (operatorStack[operatorStack.length - 1] === '(' || operatorStack[operatorStack.length - 1] === ')') {
        throw new Error("Unmatched parenthesis");
      }
      output.push(operatorStack.pop());
    }
    
    return { success: true, postfix: output.join(' ') };
  } catch (error) {
    return { success: false, errors: [error.message] };
  }
};

export const getAvailableFunctions = () => {
  return Object.keys(funcArity);
};

export const getAvailableOperators = () => {
  return Object.keys(binaryOpsPrecedence);
};

export const getFunctionArity = (functionName) => {
  return funcArity[functionName] || 0;
};

export const validatePostfixExpression = (expression) => {
  const errors = [];
  
  try {
    const tokens = expression.trim().split(/\s+/);
    
    // Check for empty expression
    if (tokens.length === 0 || (tokens.length === 1 && tokens[0] === '')) {
      errors.push("Expression cannot be empty");
      return { isValid: false, errors };
    }
    
    // Check for invalid tokens
    for (const token of tokens) {
      if (!isOperator(token) && !isFunction(token) && !isNumber(token) && !isVariable(token)) {
        errors.push(`Invalid token: ${token}`);
      }
    }
    
    // Validate postfix expression structure
    const structureErrors = validatePostfixStructure(tokens);
    errors.push(...structureErrors);
    
    return { isValid: errors.length === 0, errors };
  } catch (error) {
    return { isValid: false, errors: [error.message] };
  }
};

const validatePostfixStructure = (tokens) => {
  const errors = [];
  const stack = [];
  
  for (const token of tokens) {
    if (isNumber(token) || isVariable(token)) {
      stack.push(token);
    } else if (isOperator(token)) {
      if (stack.length < 2) {
        errors.push(`Operator '${token}' requires 2 operands, but only ${stack.length} available`);
      } else {
        stack.pop();
        stack.pop();
        stack.push('result');
      }
    } else if (isFunction(token)) {
      const arity = getFunctionArity(token);
      if (stack.length < arity) {
        errors.push(`Function '${token}' requires ${arity} operand(s), but only ${stack.length} available`);
      } else {
        for (let i = 0; i < arity; i++) {
          stack.pop();
        }
        stack.push('result');
      }
    }
  }
  
  if (stack.length !== 1) {
    errors.push(`Expression should result in exactly one value, but got ${stack.length}`);
  }
  
  return errors;
};

export const convertPostfixToInfix = (expression) => {
  const validation = validatePostfixExpression(expression);
  if (!validation.isValid) {
    return { success: false, errors: validation.errors };
  }
  
  try {
    const tokens = expression.trim().split(/\s+/);
    const stack = [];
    
    for (const token of tokens) {
      if (isNumber(token) || isVariable(token)) {
        stack.push(token);
      } else if (isOperator(token)) {
        if (stack.length < 2) {
          throw new Error(`Operator '${token}' requires 2 operands`);
        }
        const operand2 = stack.pop();
        const operand1 = stack.pop();
        
        // Add parentheses for clarity and correct precedence
        const infixExpr = `(${operand1} ${token} ${operand2})`;
        stack.push(infixExpr);
      } else if (isFunction(token)) {
        const arity = getFunctionArity(token);
        if (stack.length < arity) {
          throw new Error(`Function '${token}' requires ${arity} operand(s)`);
        }
        
        const operands = [];
        for (let i = 0; i < arity; i++) {
          operands.unshift(stack.pop());
        }
        
        const infixExpr = `${token}(${operands.join(', ')})`;
        stack.push(infixExpr);
      }
    }
    
    if (stack.length !== 1) {
      throw new Error("Invalid postfix expression");
    }
    
    return { success: true, infix: stack[0] };
  } catch (error) {
    return { success: false, errors: [error.message] };
  }
};

export const convertInfixToLatex = (expression) => {
  if (!expression) return '';
  
  let latex = expression;
  
  // First handle negative numbers to prevent interference with operators
  latex = latex.replace(/-(\d+(\.\d+)?)/g, '(-$1)');
  
  // Handle functions with word boundaries and lookahead for parentheses
  const functionNames = Object.keys(funcArity);
  functionNames.forEach(func => {
    // Create a regex that matches the function name only when followed by parentheses
    const regex = new RegExp(`\\b(${func})\\s*\\(`, 'g');
    latex = latex.replace(regex, (match, funcName) => {
      // Replace underscores with LaTeX underscore without extra escaping
      const escapedFunc = funcName.replace(/_/g, '_');
      return `\\text{${escapedFunc}}(`;
    });
  });
  
  // Replace operators with LaTeX equivalents
  latex = latex.replace(/\*/g, ' \\times ');
  latex = latex.replace(/\//g, ' \\div ');
  latex = latex.replace(/\^/g, '^');
  latex = latex.replace(/>=/g, ' \\geq ');
  latex = latex.replace(/<=/g, ' \\leq ');
  latex = latex.replace(/==/g, ' = ');
  latex = latex.replace(/&/g, ' \\land ');
  latex = latex.replace(/\|/g, ' \\lor ');
  
  // Handle variables with underscores
  latex = latex.replace(/\b[a-zA-Z][a-zA-Z0-9_]*\b(?!\()/g, (match) => {
    if (match.includes('_')) {
      // Replace underscores with LaTeX underscore without extra escaping
      return `\\text{${match.replace(/_/g, '_')}}`;
    }
    return match;
  });
  
  // Clean up any double wrapping and spacing
  latex = latex.replace(/\\text\{\\text\{([^}]+)\}\}/g, '\\text{$1}');
  latex = latex.replace(/\s+/g, ' ');
  latex = latex.trim();
  
  return latex;
}; 