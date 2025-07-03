import React, { useEffect, useState } from 'react';

const SuperResolutionImagesTable = () => {
  const [superResolutionImages, setSuperResolutionImages] = useState([]);

  useEffect(() => {
    fetch('/get-super-resolution-images')
      .then(res => res.json())
      .then(data => {
        setSuperResolutionImages(data || []);
      })
      .catch(err => console.error('❌ Error fetching super-resolution images:', err));
  }, []);

  return (
    <div>
      <h2>🖼️ Super-Resolution Images</h2>
      <table border="1" cellPadding="8">
        <thead>
          <tr>
            <th>Original Image URL</th>
            <th>Upscaled Image URL</th>
            <th>Topic Name</th>
          </tr>
        </thead>
        <tbody>
          {superResolutionImages.map((image, idx) => {
            console.log("🧪 image:", image);
            return (
              <tr key={idx}>
                <td>
                  {image.imageURL ? (
                    <a
                      href={image.imageURL}
                      target="_blank"
                      rel="noreferrer"
                      download
                    >
                      Download Original
                    </a>
                  ) : (
                    <span>N/A</span>
                  )}
                </td>
                <td>
                  {image.upscaledURL ? (
                    <a
                      href={image.upscaledURL}
                      target="_blank"
                      rel="noreferrer"
                      download
                    >
                      Download Upscaled
                    </a>
                  ) : (
                    <span>Pending</span>
                  )}
                </td>
                <td>{image.name || 'Unknown'}</td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
};

export default SuperResolutionImagesTable;
