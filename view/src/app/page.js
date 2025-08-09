'use client'

import { useState, useEffect } from "react";

export default function Home() {
  const [files, setFiles] = useState([]);
  const [uploading, setUploading] = useState(false);
  const [loading, setLoading] = useState(false);
  const [dragOver, setDragOver] = useState(false);
  const [stats, setStats] = useState(null);
  const [statsLoading, setStatsLoading] = useState(false);

  const API_BASE = "http://localhost:8080/api/fileio";

  // Fetch files from the API
  const fetchFiles = async () => {
    setLoading(true);
    try {
      const response = await fetch(`${API_BASE}/list`);
      const data = await response.json();
      setFiles(data.files || []);
    } catch (error) {
      console.error("Failed to fetch files:", error);
      alert("Failed to fetch files. Make sure the Go server is running on port 8080.");
    } finally {
      setLoading(false);
    }
  };

  // Fetch stats from the API
  const fetchStats = async () => {
    setStatsLoading(true);
    try {
      const response = await fetch(`${API_BASE}/stats`);
      const data = await response.json();
      setStats(data);
    } catch (error) {
      console.error("Failed to fetch stats:", error);
    } finally {
      setStatsLoading(false);
    }
  };

  // Upload file to the API
  const uploadFile = async (file) => {
    setUploading(true);
    const formData = new FormData();
    formData.append("file", file);

    try {
      const response = await fetch(`${API_BASE}/upload`, {
        method: "POST",
        body: formData,
      });
      
      if (response.ok) {
        const result = await response.json();
        alert(`File "${result.filename}" uploaded successfully! Size: ${result.size || result.original_size} bytes`);
        fetchFiles();
        fetchStats();
      } else {
        const error = await response.json();
        alert(`Upload failed: ${error.error}`);
      }
    } catch (error) {
      console.error("Upload error:", error);
      alert("Upload failed. Make sure the Go server is running on port 8080.");
    } finally {
      setUploading(false);
    }
  };

  // Download file
  const downloadFile = (filename) => {
    const downloadUrl = `${API_BASE}/download/${encodeURIComponent(filename)}`;
    window.open(downloadUrl, '_blank');
  };

  // Handle file input change
  const handleFileChange = (event) => {
    const selectedFile = event.target.files[0];
    if (selectedFile) {
      uploadFile(selectedFile);
    }
  };

  // Handle drag and drop
  const handleDragOver = (e) => {
    e.preventDefault();
    setDragOver(true);
  };

  const handleDragLeave = (e) => {
    e.preventDefault();
    setDragOver(false);
  };

  const handleDrop = (e) => {
    e.preventDefault();
    setDragOver(false);
    const droppedFile = e.dataTransfer.files[0];
    if (droppedFile) {
      uploadFile(droppedFile);
    }
  };

  // Format file size
  const formatFileSize = (bytes) => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const formatPercentage = (value) => {
    if (value === null || value === undefined || isNaN(value)) return '-';
    return (value * 100).toFixed(2) + '%';
  };

  // Format date
  const formatDate = (dateString) => {
    return new Date(dateString).toLocaleString();
  };

  // Load files and stats on component mount
  useEffect(() => {
    fetchFiles();
    fetchStats();
  }, []);

  return (
    <div className="min-h-screen bg-gray-50 p-8">
      <div className="max-w-7xl mx-auto">{/* widened */}
        <h1 className="text-3xl font-bold text-gray-900 mb-8">File Manager</h1>

        {/* Extended Stats Section */}
        <div className="grid md:grid-cols-4 gap-6 mb-8">
          {/* existing cards */}
          <div className="bg-white rounded-lg shadow p-6">
            <h3 className="text-sm font-medium text-gray-500">Files</h3>
            <p className="mt-2 text-2xl font-semibold text-gray-900">{statsLoading ? '…' : (stats ? stats.file_count : 0)}</p>
            <p className="mt-1 text-xs text-gray-500">Unique Hashes: {statsLoading || !stats ? '…' : stats.unique_hash_count}</p>
          </div>
          <div className="bg-white rounded-lg shadow p-6">
            <h3 className="text-sm font-medium text-gray-500">Original vs Compressed</h3>
            <p className="mt-2 text-sm text-gray-700">
              {statsLoading || !stats ? '…' : `${formatFileSize(stats.total_original_size)} → ${formatFileSize(stats.total_compressed_size)}`}
            </p>
            <p className="mt-1 text-xs text-gray-500">Saved: {statsLoading || !stats ? '…' : formatFileSize(stats.space_saved)}</p>
          </div>
          <div className="bg-white rounded-lg shadow p-6">
            <h3 className="text-sm font-medium text-gray-500">Physical Usage</h3>
            <p className="mt-2 text-sm text-gray-700">{statsLoading || !stats ? '…' : formatFileSize(stats.physical_objects_size || 0)}</p>
            <p className="mt-1 text-xs text-gray-500">Blobs: {statsLoading || !stats ? '…' : stats.physical_objects_count}</p>
          </div>
          <div className="bg-white rounded-lg shadow p-6">
            <h3 className="text-sm font-medium text-gray-500">Compression Ratio</h3>
            <p className="mt-2 text-2xl font-semibold text-gray-900">{statsLoading || !stats ? '…' : (stats.compression_ratio ? (stats.compression_ratio).toFixed(2) : '1.00')}</p>
            <p className="mt-1 text-xs text-gray-500">Space Saved %: {statsLoading || !stats ? '…' : (stats.space_saved_percentage ? stats.space_saved_percentage.toFixed(2) + '%' : '0%')}</p>
          </div>
        </div>

        {stats && (
          <div className="grid md:grid-cols-2 gap-6 mb-8">
            <div className="bg-white rounded-lg shadow p-6">
              <h3 className="text-sm font-medium text-gray-500 mb-4">Dedup Savings</h3>
              <ul className="space-y-2 text-sm text-gray-700">
                <li>Logical Compressed: {formatFileSize(stats.total_compressed_size || 0)}</li>
                <li>Physical Compressed: {formatFileSize(stats.physical_objects_size || 0)}</li>
                <li>Dedup Saved (Compressed): {formatFileSize(stats.dedup_saved_compressed || 0)} ({stats.dedup_saved_compr_pct ? stats.dedup_saved_compr_pct.toFixed(2) + '%' : '0%'})</li>
                <li>Dedup Saved (Original Basis): {formatFileSize(stats.dedup_saved_original || 0)} ({stats.dedup_saved_original_pct ? stats.dedup_saved_original_pct.toFixed(2) + '%' : '0%'})</li>
              </ul>
            </div>
            <div className="bg-white rounded-lg shadow p-6">
              <h3 className="text-sm font-medium text-gray-500 mb-4">Compression & MIME Types</h3>
              <div className="mb-3">
                <p className="text-xs font-semibold text-gray-600 mb-1">Compression</p>
                <div className="flex flex-wrap gap-2">
                  {stats.compression_types && Object.entries(stats.compression_types).map(([k,v]) => (
                    <span key={k} className="px-2 py-1 bg-blue-50 text-blue-700 rounded text-xs">{k || 'unknown'}: {v}</span>
                  ))}
                </div>
              </div>
              <div>
                <p className="text-xs font-semibold text-gray-600 mb-1">MIME</p>
                <div className="flex flex-wrap gap-2">
                  {stats.mime_types && Object.entries(stats.mime_types).map(([k,v]) => (
                    <span key={k} className="px-2 py-1 bg-purple-50 text-purple-700 rounded text-xs">{k || 'unknown'}: {v}</span>
                  ))}
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Upload Section */}
        <div className="bg-white rounded-lg shadow-md p-6 mb-8">
          <h2 className="text-xl font-semibold text-gray-800 mb-4">Upload Files</h2>
          
          {/* Drag and Drop Zone */}
          <div
            className={`border-2 border-dashed rounded-lg p-8 text-center transition-colors ${
              dragOver
                ? "border-blue-500 bg-blue-50"
                : "border-gray-300 hover:border-gray-400"
            }`}
            onDragOver={handleDragOver}
            onDragLeave={handleDragLeave}
            onDrop={handleDrop}
          >
            <div className="text-gray-600">
              <svg className="mx-auto h-12 w-12 text-gray-400 mb-4" stroke="currentColor" fill="none" viewBox="0 0 48 48">
                <path d="M28 8H12a4 4 0 00-4 4v20m32-12v8m0 0v8a4 4 0 01-4 4H12a4 4 0 01-4-4v-4m32-4l-3.172-3.172a4 4 0 00-5.656 0L28 28M8 32l9.172-9.172a4 4 0 015.656 0L28 28m0 0l4 4m4-24h8m-4-4v8m-12 4h.02" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" />
              </svg>
              <p className="text-lg">Drag and drop a file here, or</p>
              <label className="inline-block mt-2 px-4 py-2 bg-blue-600 text-white rounded-md cursor-pointer hover:bg-blue-700 transition-colors">
                Choose File
                <input
                  type="file"
                  className="hidden"
                  onChange={handleFileChange}
                  disabled={uploading}
                />
              </label>
            </div>
          </div>
          
          {uploading && (
            <div className="mt-4 text-center">
              <div className="inline-flex items-center text-blue-600">
                <svg className="animate-spin -ml-1 mr-3 h-5 w-5" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
                Uploading...
              </div>
            </div>
          )}
        </div>

        {/* File List Section */}
        <div className="bg-white rounded-lg shadow-md">
          <div className="p-6 border-b border-gray-200">
            <div className="flex justify-between items-center">
              <h2 className="text-xl font-semibold text-gray-800">Files</h2>
              <div className="flex gap-2">
                <button
                  onClick={() => { fetchFiles(); fetchStats(); }}
                  disabled={loading || statsLoading}
                  className="px-4 py-2 bg-green-600 text-white rounded-md hover:bg-green-700 transition-colors disabled:opacity-50"
                >
                  {(loading || statsLoading) ? "Refreshing..." : "Refresh"}
                </button>
              </div>
            </div>
          </div>
          
          <div className="p-6">
            {loading ? (
              <div className="text-center py-8">
                <svg className="animate-spin mx-auto h-8 w-8 text-gray-400" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
                <p className="mt-2 text-gray-600">Loading files...</p>
              </div>
            ) : files.length === 0 ? (
              <div className="text-center py-8">
                <svg className="mx-auto h-12 w-12 text-gray-400 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
                </svg>
                <p className="text-gray-600">No files uploaded yet</p>
                <p className="text-sm text-gray-500 mt-1">Upload your first file using the form above</p>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-200">
                  <thead className="bg-gray-50">
                    <tr>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Filename
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Size
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Upload Date
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Actions
                      </th>
                    </tr>
                  </thead>
                  <tbody className="bg-white divide-y divide-gray-200">
                    {files.map((file) => (
                      <tr key={file.id} className="hover:bg-gray-50">
                        <td className="px-6 py-4 whitespace-nowrap">
                          <div className="flex items-center">
                            <svg className="h-5 w-5 text-gray-400 mr-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                            </svg>
                            <span className="text-sm font-medium text-gray-900">{file.filename}</span>
                          </div>
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                          {formatFileSize(file.size)}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                          {formatDate(file.created_at)}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                          <button
                            onClick={() => downloadFile(file.filename)}
                            className="text-blue-600 hover:text-blue-900 transition-colors"
                          >
                            Download
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>

        {/* Footer */}
        <div className="mt-8 text-center text-gray-500 text-sm">
          <p>File Manager powered by Go4Pack API</p>
          <p className="mt-1">Make sure your Go server is running on <code className="bg-gray-200 px-1 rounded">localhost:8080</code></p>
        </div>
      </div>
    </div>
  );
}
