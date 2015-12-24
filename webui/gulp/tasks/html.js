'use strict';

// Scan Your HTML For Assets & Optimize Them
module.exports = function (gulp, plugins, config) { return function () {
  return gulp.src(['app/**/*.html', '.tmp/*.html', 'app/{elements,test}/**/*.html'])
    .pipe(plugins.useref({searchPath: ['.tmp', 'app', 'dist']}))
    // Concatenate And Minify JavaScript
    .pipe(plugins.if('*.js', plugins.uglify({preserveComments: 'some'})))
    // Concatenate And Minify Styles
    // In case you are still using useref build blocks
    .pipe(plugins.if('*.css', plugins.cssmin()))
    // Add shim-shadowdom to link with main.css
    .pipe(plugins.if('*.html', plugins.replace(
      'main.css">', 'main.css" shim-shadowdom>')))
    // Minify Any HTML
    .pipe(plugins.if('*.html', plugins.minifyHtml({
      empty: true,  // KEEP empty attributes
      loose: true,  // KEEP one whitespace
      quotes: true, // KEEP arbitrary quotes
      spare: true   // KEEP redundant attributes
    })))
    // Output Files
    .pipe(gulp.dest('dist'))
    .pipe(plugins.size({title: 'html'}));
};};
