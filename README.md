<h3> 
Classification of Three-dimensional Geometric Objects
</h3>
<p>
This is a web application written in Go that makes use of the html/template package to dynamically
create the web page. Start the web server at bin\geometricClassify.exe and connect to it from your web browser
at http://127.0.0.1:8080/geometricclassification. 
Geometric Classification classifies the internal structure of 3D geometric objects
such as ellipsoids, parabloids, cubes, boxes, planes, lemniscates, cardiods, four-leaf rose, or cones.
It slices thegeometric objects along axial planes in the Cartesian coordinate system.
The object can be solids as well as surfaces.
It gives an overview of the planes in i, j, k axes along with the option
of zooming in on a particular axial plane.  It is possible to select and
view particular planes in the geometric object with different step sizes.
It will classify the geometric object and display the results.  It does this
by comparing the noisy test samples that are displaced randomly in space with
references of the geometric objects that are noise free and centered.  The metrics
are mass sums of the rows and columns of plane in each axes in the Cartesian
coordinate system.  The least square error determines how the sample is classified.
The difference between the reference class mass sums and the test sample is the error.
</p>

<p>
Since the classification involves 3-dimensional searches, it takes a long time to 
finish execution.  For classification of 100 samples of the 19 geometric objects,
it took about 4.5 hours.  The geometric objects are represented by densities which
can be any concentration quantity.  By summing up the row or column densities you
can obtain a quantity proportional to mass.	Thus the row sums and columns sums 
distinguish different geometric objects and can be used to classify the object.
</p>

<h4>Classification of 100 samples, no noise, 19 geometric objects.</h4>
<img width="1386" height="898" alt="image" src="https://github.com/user-attachments/assets/69afb153-8394-459e-9219-cc1863767183" />
<h4>Classification of 100 samples, level 2 noise, 19 geometric objects.</h4>
<img width="1407" height="898" alt="image" src="https://github.com/user-attachments/assets/41fbdfea-6279-4901-884a-58a48cf073d5" />
<h4>Classification of 100 samples, level 4 noise, 19 geometric objects.</h4>
<img width="1375" height="897" alt="image" src="https://github.com/user-attachments/assets/c44429c6-a0f3-48da-a98b-8f278c098500" />
