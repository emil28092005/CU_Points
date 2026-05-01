import { type HTMLAttributes } from 'react';

interface CardProps extends HTMLAttributes<HTMLDivElement> {
  children: React.ReactNode;
}

export function Card({ children, className = '', ...props }: CardProps) {
  return (
    <div
      className={`rounded-2xl bg-gray-800 p-6 shadow-sm ring-1 ring-gray-700 ${className}`}
      {...props}
    >
      {children}
    </div>
  );
}
