// src/constants/dataTypes.js

export const dataTypes = [
    'FP8E5M2', 'FP8E4M3', 'FP16', 'FP32', 'FP64',
    'Int8', 'Int16', 'Int32', 'Int64',
    'Uint8', 'Uint16', 'Uint32', 'Uint64',
    'String', 'Bool',
    'FP8E5M2Vector', 'FP8E4M3Vector', 'FP16Vector', 'FP32Vector', 'FP64Vector',
    'Int8Vector', 'Int16Vector', 'Int32Vector', 'Int64Vector',
    'Uint8Vector', 'Uint16Vector', 'Uint32Vector', 'Uint64Vector',
    'StringVector', 'BoolVector'
  ];
  
  export const addDataTypePrefix = (value) => `DataType${value}`;
  
  export const removeDataTypePrefix = (value) => value.replace('DataType', '');