import React, { useRef, useState, useCallback } from 'react';
import { Camera, RefreshCw, Upload, Check, AlertCircle } from 'lucide-react';
import Button from '../../components/button';
import { apiFetch } from '../../utils/api';
import { useAuth } from '../../contexts/AuthContext';

const CameraCapturePage: React.FC = () => {
  const { token } = useAuth();
  const videoRef = useRef<HTMLVideoElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  
  const [stream, setStream] = useState<MediaStream | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isCapturing, setIsCapturing] = useState(false);
  const [capturedImage, setCapturedImage] = useState<string | null>(null);
  const [isUploading, setIsUploading] = useState(false);
  const [uploadSuccess, setUploadSuccess] = useState(false);

  const startCamera = async () => {
    try {
      setError(null);
      const mediaStream = await navigator.mediaDevices.getUserMedia({ 
        video: { facingMode: 'environment', width: { ideal: 1920 }, height: { ideal: 1080 } } 
      });
      setStream(mediaStream);
      if (videoRef.current) {
        videoRef.current.srcObject = mediaStream;
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not access camera. Please grant permissions.');
    }
  };

  const stopCamera = useCallback(() => {
    if (stream) {
      stream.getTracks().forEach(track => track.stop());
      setStream(null);
    }
  }, [stream]);

  // Clean up on unmount
  React.useEffect(() => {
    return stopCamera;
  }, [stopCamera]);

  const capturePhoto = () => {
    if (videoRef.current && canvasRef.current) {
      setIsCapturing(true);
      const video = videoRef.current;
      const canvas = canvasRef.current;
      
      canvas.width = video.videoWidth;
      canvas.height = video.videoHeight;
      
      const ctx = canvas.getContext('2d');
      if (ctx) {
        ctx.drawImage(video, 0, 0, canvas.width, canvas.height);
        // Convert to jpeg blob
        const dataUrl = canvas.toDataURL('image/jpeg', 0.9);
        setCapturedImage(dataUrl);
        stopCamera();
      }
      setIsCapturing(false);
    }
  };

  const resetCapture = () => {
    setCapturedImage(null);
    setUploadSuccess(false);
    startCamera();
  };

  const uploadPhoto = async () => {
    if (!capturedImage) return;
    
    setIsUploading(true);
    setError(null);
    
    try {
      // Convert base64 to blob
      const res = await fetch(capturedImage);
      const blob = await res.blob();
      
      // We will send it to the Go API
      // Since the backend handles MQTT directly from python, 
      // but HTTP uploads use POST /photos
      const formData = new FormData();
      formData.append('file', blob, 'capture.jpg');
      formData.append('device_id', 'browser-camera');
      
      const response = await apiFetch('/photos', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`
        },
        body: formData
      });
      
      if (response.ok) {
        setUploadSuccess(true);
      } else {
        const data = await response.json().catch(() => ({}));
        setError(data.message || 'Upload failed');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Network error during upload');
    } finally {
      setIsUploading(false);
    }
  };

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      <div className="bg-white p-6 rounded-xl shadow-sm border border-sky-100">
        <div className="flex items-center justify-between mb-6">
          <div>
            <h2 className="text-2xl font-bold text-slate-800">Camera Capture</h2>
            <p className="text-slate-500">Capture documents directly from your browser.</p>
          </div>
          {!stream && !capturedImage && (
            <Button 
              text="Start Camera" 
              onClick={startCamera} 
              variant="primary"
            />
          )}
        </div>

        {error && (
          <div className="mb-6 p-4 bg-red-50 text-red-700 rounded-lg flex items-center">
            <AlertCircle className="w-5 h-5 mr-2" />
            {error}
          </div>
        )}

        {uploadSuccess && (
          <div className="mb-6 p-4 bg-emerald-50 text-emerald-700 rounded-lg flex items-center">
            <Check className="w-5 h-5 mr-2" />
            Photo uploaded successfully! It will be processed by the OCR pipeline.
          </div>
        )}

        <div className="relative aspect-video bg-slate-100 rounded-lg overflow-hidden border border-slate-200">
          {!stream && !capturedImage && (
            <div className="absolute inset-0 flex flex-col items-center justify-center text-slate-400">
              <Camera className="w-16 h-16 mb-4 opacity-50" />
              <p>Camera is inactive</p>
            </div>
          )}
          
          <video 
            ref={videoRef}
            autoPlay 
            playsInline 
            className={`w-full h-full object-cover ${!stream || capturedImage ? 'hidden' : ''}`}
          />
          
          <canvas ref={canvasRef} className="hidden" />
          
          {capturedImage && (
            <img 
              src={capturedImage} 
              alt="Captured document" 
              className="w-full h-full object-contain bg-slate-900"
            />
          )}
        </div>

        <div className="mt-6 flex justify-center space-x-4">
          {stream && !capturedImage && (
            <Button 
              text={isCapturing ? "Capturing..." : "Capture Document"} 
              onClick={capturePhoto}
              variant="primary"
              size="lg"
            />
          )}
          
          {capturedImage && !uploadSuccess && (
            <>
              <Button 
                text="Retake" 
                onClick={resetCapture}
                variant="outline"
              />
              <button 
                onClick={uploadPhoto}
                disabled={isUploading}
                className="px-4 py-2 bg-sky-600 text-white rounded-lg hover:bg-sky-700 font-medium transition-colors flex items-center disabled:opacity-50"
              >
                {isUploading ? <RefreshCw className="w-5 h-5 mr-2 animate-spin" /> : <Upload className="w-5 h-5 mr-2" />}
                {isUploading ? "Uploading..." : "Upload Document"}
              </button>
            </>
          )}
          
          {uploadSuccess && (
            <Button 
              text="Capture Another" 
              onClick={resetCapture}
              variant="primary"
            />
          )}
        </div>
      </div>
    </div>
  );
};

export default CameraCapturePage;
