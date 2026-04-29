import { Loader2Icon } from 'lucide-react'

import { cn } from '@/lib/utils'

type SpinnerProps = React.ComponentProps<'svg'> & {
  size?: 'sm' | 'default' | 'lg'
}

function Spinner({
  className,
  size = 'default',
  ...props
}: SpinnerProps) {
  const sizeClass =
    size === 'sm' ? 'size-4' : size === 'lg' ? 'size-6' : 'size-4'

  return (
    <Loader2Icon
      role="status"
      aria-label="Loading"
      className={cn(sizeClass, 'animate-spin', className)}
      {...props}
    />
  )
}

export { Spinner }
