import { useState, useRef, useEffect, type ReactNode } from 'react';
import './ResizableSidebar.css';

interface ResizableSidebarProps {
  children: ReactNode;
  defaultWidth?: number;
  minWidth?: number;
}

export function ResizableSidebar({
  children,
  defaultWidth = 350,
  minWidth = 300,
}: ResizableSidebarProps) {
  const [width, setWidth] = useState(defaultWidth);
  const [isResizing, setIsResizing] = useState(false);
  const [maxWidth, setMaxWidth] = useState(() => window.innerWidth - 100);
  const sidebarRef = useRef<HTMLDivElement>(null);

  // Handle window resize to update max width
  useEffect(() => {
    const handleWindowResize = () => {
      const newMaxWidth = window.innerWidth - 100;
      setMaxWidth(newMaxWidth);

      // Constrain current width if it exceeds new max
      setWidth(prevWidth => Math.min(prevWidth, newMaxWidth));
    };

    window.addEventListener('resize', handleWindowResize);

    return () => {
      window.removeEventListener('resize', handleWindowResize);
    };
  }, []);

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isResizing) return;

      const newWidth = e.clientX;

      if (newWidth >= minWidth && newWidth <= maxWidth) {
        setWidth(newWidth);
      }
    };

    const handleMouseUp = () => {
      setIsResizing(false);
    };

    if (isResizing) {
      document.addEventListener('mousemove', handleMouseMove);
      document.addEventListener('mouseup', handleMouseUp);
      document.body.style.cursor = 'col-resize';
      document.body.style.userSelect = 'none';
    }

    return () => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
    };
  }, [isResizing, minWidth, maxWidth]);

  const handleMouseDown = () => {
    setIsResizing(true);
  };

  return (
    <div className="resizable-sidebar-container">
      <div
        ref={sidebarRef}
        className="sidebar"
        style={{ width: `${width}px` }}
      >
        {children}
      </div>
      <div
        className={`resize-handle ${isResizing ? 'resizing' : ''}`}
        onMouseDown={handleMouseDown}
      />
    </div>
  );
}
