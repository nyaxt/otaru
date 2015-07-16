'use strict';

// Include Gulp, Plugins & Config
var browserSync   = require('browser-sync');
var fs            = require('fs');
var gulp          = require('gulp');
var plugins       = require('gulp-load-plugins')();
var yaml          = require('js-yaml');

var configFile = './gulp-tasks-config.yml';
var config = yaml.safeLoad(fs.readFileSync(configFile, 'utf-8'));

// Get a task from the tasks directory with default parameters
function getTask(task) {
  return require('./gulp/tasks/' + task)(gulp, plugins, config);
}

// Tasks
// -----

// Copy Web Fonts To Dist
gulp.task('fonts', getTask('fonts'));

// Scan Your HTML For Assets & Optimize Them
gulp.task('html', getTask('html'));

// Optimize Images
gulp.task('images', getTask('images'));

// Inject bower components
gulp.task('wiredep', getTask('wiredep'));


// Serve Tasks
// -----------

// Watch Files For Changes & Reload
gulp.task('serve:base',
  require('./gulp/tasks/serve')(gulp, config, browserSync));

gulp.task('serve', [
    'fonts'
  ], function (cb) {
    require('run-sequence')(
      'serve:base',
      cb
    );
  }
);

// Build and serve the output from the dist build
gulp.task('serve:dist', ['default'],
  require('./gulp/tasks/serve-dist')(gulp, config, browserSync));


// Build Task and Subtasks
// -----------------------

// Get gzipped size of build
gulp.task('build-size', getTask('build-size'));

// Clean Output Directory
gulp.task('clean', require('del').bind(null, ['.tmp', 'dist']));

// Copy All Files At The Root Level (app)
gulp.task('copy', getTask('copy'));

// Copy files only for build a element
gulp.task('copy-build-element', getTask('copy-build-element'));

// Gzip text files
gulp.task('gzip', getTask('gzip'));

// Vulcanize imports
gulp.task('vulcanize', getTask('vulcanize'));

// Babel js
gulp.task('babel', getTask('babel'));

// Build Production Files, the Default Task
gulp.task('default', ['clean'], function (cb) {
  require('run-sequence')(
    ['copy'],
    ['images', 'fonts', 'html'],
    'vulcanize',
    'babel',
    cb);
});

// Deploy Tasks
// ------------


// Tool Tasks
// -----------


// Test Tasks
// ----------

// Load tasks for web-component-tester
// Adds tasks for `gulp test:local` and `gulp test:remote`
try { require('web-component-tester').gulp.init(gulp); } catch (err) {}
