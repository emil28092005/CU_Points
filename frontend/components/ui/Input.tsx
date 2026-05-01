import { type InputHTMLAttributes, useId } from 'react';

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label: string;
  error?: string;
}

export function Input({ label, error, className = '', ...props }: InputProps) {
  const id = useId();
  return (
    <div className="flex flex-col gap-1">
      <label htmlFor={id} className="text-sm font-medium text-gray-300">
        {label}
      </label>
      <input
        id={id}
        className={`rounded-lg border bg-gray-700 px-3 py-2 text-sm text-white placeholder-gray-500 outline-none transition-colors focus:ring-2 ${error ? 'border-red-500 focus:border-red-500 focus:ring-red-800' : 'border-gray-600 focus:border-blue-500 focus:ring-blue-900'} ${className}`}
        {...props}
      />
      {error && <p className="text-xs text-red-400">{error}</p>}
    </div>
  );
}
