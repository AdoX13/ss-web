import React from 'react';

interface DeviceCardProps {
  deviceId: string;
  deviceName: string;
  onCaptureClick?: () => void;
  onStartLiveClick?: () => void;
  onStopLiveClick?: () => void;
}

const noop = () => {};

const DeviceCard: React.FC<DeviceCardProps> = ({
  deviceId,
  deviceName,
  onCaptureClick = noop,
  onStartLiveClick = noop,
  onStopLiveClick = noop,
}) => {
  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg shadow-md border border-transparent dark:border-gray-700 overflow-hidden transition-all hover:shadow-lg p-4">
      <h3 className="text-lg font-medium text-gray-800 dark:text-gray-100 mb-4">
        {deviceId} - {deviceName}
      </h3>
      <div className="flex gap-2 flex-wrap">
        <button
          onClick={onCaptureClick}
          className="flex-1 px-3 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 dark:focus:ring-offset-gray-800 transition-colors text-sm"
        >
          Capture
        </button>
        <button
          onClick={onStartLiveClick}
          className="flex-1 px-3 py-2 bg-emerald-600 text-white rounded-md hover:bg-emerald-700 focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:ring-offset-2 dark:focus:ring-offset-gray-800 transition-colors text-sm"
        >
          Start Live
        </button>
        <button
          onClick={onStopLiveClick}
          className="flex-1 px-3 py-2 bg-red-600 text-white rounded-md hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-offset-2 dark:focus:ring-offset-gray-800 transition-colors text-sm"
        >
          Stop Live
        </button>
      </div>
    </div>
  );
};

export default DeviceCard;
